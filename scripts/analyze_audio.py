#!/usr/bin/env python3
"""
Sonara Audio Analyzer
Performs FFT analysis on audio files with FIFINE K669 microphone calibration
"""

# Enable logging to stderr (visible to Go process) - MUST be first
import sys
import logging
logging.basicConfig(level=logging.INFO, format='[PYTHON] %(asctime)s - %(levelname)s - %(message)s', stream=sys.stderr)
import json
import math
import numpy as np
from scipy import signal
from scipy import interpolate
from scipy.io import wavfile
import librosa

from reference_signals import get_reference_manager


class FrequencyAnalyzer:
    """
    Professional frequency response analyzer using sweep deconvolution.

    Performs sine sweep deconvolution for accurate acoustic measurements.
    """

    def __init__(self):
        self.fft_size = 32768  # 32k FFT for high resolution
        self.smoothing_fraction = 1/3  # 1/6 octave smoothing
        self.reference_freq = 1000  # 1kHz normalization
        self.ref_manager = get_reference_manager()

    def align_signals(self, recorded, reference):
        """
        Align recorded signal with reference using cross-correlation.

        Compensates for recording delays by finding the optimal alignment
        between the recorded signal and reference sweep.

        Args:
            recorded: Recorded audio signal (numpy array)
            reference: Reference sweep signal (numpy array)

        Returns:
            numpy array: Aligned recorded signal
        """
        logging.info("Aligning signals using cross-correlation")

        # Compute cross-correlation
        correlation = signal.correlate(recorded, reference, mode='valid')

        # Find the delay that gives maximum correlation
        delay = np.argmax(np.abs(correlation))
        logging.info(f"Optimal delay found: {delay} samples")

        # Align by trimming the recorded signal
        if delay < len(recorded):
            aligned = recorded[delay:delay + len(reference)]
            logging.info(f"Aligned signal length: {len(aligned)} samples")
            return aligned
        else:
            # If delay is too large, return original (fallback)
            logging.warning("Alignment delay too large, using original signal")
            return recorded[:len(reference)] if len(recorded) > len(reference) else recorded

    def analyze_sweep_deconvolution(self, recorded_file, signal_id, room_data=None):
        """
        Main sweep deconvolution pipeline for frequency response measurement.

        Args:
            recorded_file: Path to recorded audio file
            signal_id: Identifier for reference signal (e.g., "sine_sweep_20_20k")
            room_data: Optional dict with room dimensions in feet (room_length_feet, room_width_feet, room_height_feet)

        Returns:
            tuple: (frequencies, response_db, room_modes) - frequencies and response arrays, plus calculated room mode frequencies
        """
        logging.info(f"Starting sweep deconvolution analysis for file: {recorded_file}, signal: {signal_id}")

        # 1. Load cached reference data
        logging.info("Step 1: Loading reference signal data")
        ref_data = self.ref_manager.get_signal_data(signal_id)
        if not ref_data:
            raise ValueError(f"Unknown or invalid signal ID: {signal_id}")
        logging.info(f"Reference signal loaded successfully - sample rate: {ref_data.get('sample_rate', 'unknown')}Hz")

        # 2. Load recorded audio
        logging.info("Step 2: Loading recorded audio file")
        try:
            recorded, sr = librosa.load(recorded_file, sr=None, mono=True)
            duration = len(recorded) / sr
            logging.info(f"Recorded audio loaded - sample rate: {sr}Hz, duration: {duration:.2f}s, samples: {len(recorded)}")
        except Exception as e:
            raise ValueError(f"Failed to load recorded file {recorded_file}: {e}")

        # 3. Align signals using cross-correlation
        logging.info("Step 3: Aligning recorded signal with reference")
        aligned_recorded = self.align_signals(recorded, ref_data["sweep_signal"])
        logging.info(f"Signal alignment completed - aligned length: {len(aligned_recorded)} samples")

        # 4. Perform regularized spectral division deconvolution
        logging.info("Step 4: Performing regularized spectral division deconvolution")
        impulse_response = self.deconvolve_signals(
            aligned_recorded, ref_data["sweep_signal"]
        )
        logging.info(f"Deconvolution completed - impulse response length: {len(impulse_response)} samples")

        # 5. Extract acoustic impulse window
        logging.info("Step 5: Extracting acoustic impulse window")
        impulse_windowed = self.extract_impulse_window(impulse_response, sr)
        logging.info(f"Impulse window extracted - length: {len(impulse_windowed)} samples")

        # 6. Convert impulse response to frequency response
        logging.info("Step 6: Converting impulse to frequency response")
        freqs, response_db = self.impulse_to_frequency_response(impulse_windowed, sr)
        logging.info(f"Frequency response calculated - {len(freqs)} frequency points")

        # 7. Apply fractional octave smoothing
        logging.info("Step 7: Applying fractional octave smoothing")
        smooth_response = self.apply_fractional_octave_smoothing(freqs, response_db)
        logging.info("Fractional octave smoothing applied")

        # 8. Normalize to 0dB at 1kHz
        logging.info("Step 8: Normalizing response to 0dB at 1kHz")
        final_response = self.normalize_response(freqs, smooth_response)
        logging.info("Response normalization completed")

        # 9. Calculate room modes if room data is available
        room_modes = []
        if room_data:
            logging.info("Step 9: Calculating room modes from provided dimensions")
            room_modes = self.calculate_room_modes(room_data)
            logging.info(f"Room modes calculated: {len(room_modes)} modes found")
        else:
            logging.info("Step 9: No room data provided, skipping room mode calculation")

        logging.info(f"Sweep deconvolution analysis completed - final result: {len(freqs)} frequency points, {len(room_modes)} room modes")
        return freqs, final_response, room_modes


    def deconvolve_signals(self, recorded, reference_sweep, lambda_reg=1e-3):
        """
        Deconvolve recorded signal with reference sweep using regularized spectral division.

        Uses frequency domain division with regularization to prevent noise amplification
        where the reference sweep has weak energy.

        Args:
            recorded: Recorded audio signal (numpy array)
            reference_sweep: Reference sweep signal (numpy array)
            lambda_reg: Regularization parameter (default: 1e-3)

        Returns:
            numpy array: Impulse response
        """
        logging.info("Performing regularized spectral division deconvolution")

        # Pad to avoid circular convolution artifacts
        n = len(recorded) + len(reference_sweep) - 1

        # FFT of both signals
        Y = np.fft.fft(recorded, n)  # Recorded signal
        X = np.fft.fft(reference_sweep, n)  # Reference sweep

        # Regularized spectral division: H = (Y * conj(X)) / (|X|² + λ)
        # This gives us the transfer function H such that recorded = reference_sweep * H
        H = (Y * np.conj(X)) / (np.abs(X)**2 + lambda_reg)

        # Inverse FFT to get impulse response
        impulse = np.real(np.fft.ifft(H))

        logging.info(f"Deconvolution completed - impulse response length: {len(impulse)} samples")
        return impulse

    def extract_impulse_window(self, impulse, sample_rate):
        """
        Extract the main impulse response with acoustic-appropriate windowing.

        Finds the main peak and windows around it for room acoustic measurements.
        Uses 450ms total window (50ms before peak, 400ms after peak).
        """
        # Find main impulse peak
        peak_idx = np.argmax(np.abs(impulse))

        # Acoustic window: 450ms total (50ms before, 400ms after peak)
        pre_samples = int(0.05 * sample_rate)  # 50ms before peak (reduced from 100ms)
        post_samples = int(0.4 * sample_rate)  # 400ms after peak (unchanged)

        start = max(0, peak_idx - pre_samples)
        end = min(len(impulse), peak_idx + post_samples)

        logging.info(f"Impulse window: peak at {peak_idx}, window from {start} to {end} ({(end-start)/sample_rate*1000:.1f}ms)")
        return impulse[start:end]

    def impulse_to_frequency_response(self, impulse, sample_rate):
        """
        Convert impulse response to frequency response via FFT.
        """
        # Apply window to reduce spectral leakage
        window = signal.get_window('blackmanharris', len(impulse))
        windowed = impulse * window

        # High-resolution FFT
        fft_result = np.fft.rfft(windowed, n=self.fft_size)
        frequencies = np.fft.rfftfreq(self.fft_size, 1/sample_rate)

        # Convert to dB magnitude
        magnitude_db = 20 * np.log10(np.abs(fft_result) + 1e-12)

        # Filter to audible range (20Hz - 20kHz)
        audible_mask = (frequencies >= 20) & (frequencies <= 20000)

        return frequencies[audible_mask], magnitude_db[audible_mask]

    def apply_fractional_octave_smoothing(self, frequencies, magnitude_db):
        """
        Apply 1/12 octave smoothing for clean, detailed curves.

        Correctly averages in power domain (not dB domain) for mathematically sound results.
        Smoothing reduces noise while preserving acoustic detail.
        """
        smoothed = np.zeros_like(magnitude_db)

        for i, center_freq in enumerate(frequencies):
            if center_freq < 20:
                smoothed[i] = magnitude_db[i]
                continue

            # Calculate smoothing bandwidth (1/12 octave)
            factor = 2 ** (self.smoothing_fraction / 2)
            lower_freq = center_freq / factor
            upper_freq = center_freq * factor

            # Find all frequencies in this band
            in_band = (frequencies >= lower_freq) & (frequencies <= upper_freq)

            if np.any(in_band):
                # Convert dB to power, average power, then back to dB
                # This is mathematically correct (average intensity, not amplitudes)
                power_values = 10 ** (magnitude_db[in_band] / 10)  # dB to power
                avg_power = np.mean(power_values)  # Average power
                smoothed[i] = 10 * np.log10(avg_power + 1e-12)  # Power to dB
            else:
                smoothed[i] = magnitude_db[i]

        return smoothed

    def normalize_response(self, frequencies, response_db):
        """
        Normalize frequency response to 0dB at 1kHz (industry standard).
        """
        # Find closest frequency to 1kHz
        ref_idx = np.argmin(np.abs(frequencies - self.reference_freq))
        ref_level = response_db[ref_idx]

        # Subtract reference level from all points
        return response_db - ref_level

    def resample_log_spaced(self, frequencies, magnitudes, num_points=300):
        """
        Resample frequency response to log-spaced points for display.
        """
        # Create log-spaced frequency points from 20Hz to 20kHz
        log_freqs = np.logspace(np.log10(20), np.log10(20000), num_points)

        # Create interpolation function
        interp_func = interpolate.interp1d(
            frequencies,
            magnitudes,
            kind='linear',  # Linear is more accurate for audio
            bounds_error=False,
            fill_value=np.nan
        )

        # Interpolate at log-spaced frequencies
        log_magnitudes = interp_func(log_freqs)

        # Remove any NaN values (shouldn't happen with 20-20k range)
        valid_mask = ~np.isnan(log_magnitudes)

        return log_freqs[valid_mask], log_magnitudes[valid_mask]

    def calculate_room_modes(self, room_data, max_modes=5, min_spacing_octaves=1/6):
        """
        Calculate room modes (20–300 Hz), then keep only a few that are
        well-spaced in frequency for clearer visualization.

        - max_modes: cap on how many markers to return
        - min_spacing_octaves: minimum spacing between kept modes (e.g., 1/3 octave)
        """
        try:
            length = room_data.get('room_length_feet', room_data.get('room_length', 0))
            width = room_data.get('room_width_feet', room_data.get('room_width', 0))
            height = room_data.get('room_height_feet', room_data.get('room_height', 0))

            length_m = length * 0.3048 if length > 0 else 0
            width_m = width * 0.3048 if width > 0 else 0
            height_m = height * 0.3048 if height > 0 else 0

            if not any([length_m, width_m, height_m]):
                logging.info("No room dimensions provided, skipping room mode calculation")
                return []

            c = 343.0
            modes = []

            # First-order axial fundamentals
            if length_m > 0:
                modes.append(c / (2 * length_m))
            if width_m > 0:
                modes.append(c / (2 * width_m))
            if height_m > 0:
                modes.append(c / (2 * height_m))

            # First-order tangential
            if length_m > 0 and width_m > 0:
                modes.append(c / (2 * math.sqrt(length_m**2 + width_m**2)))
            if length_m > 0 and height_m > 0:
                modes.append(c / (2 * math.sqrt(length_m**2 + height_m**2)))
            if width_m > 0 and height_m > 0:
                modes.append(c / (2 * math.sqrt(width_m**2 + height_m**2)))

            # First-order oblique
            if all([length_m > 0, width_m > 0, height_m > 0]):
                modes.append(c / (2 * math.sqrt(length_m**2 + width_m**2 + height_m**2)))

            # Keep only 20–300 Hz as before
            modes = sorted(m for m in modes if 20 <= m <= 300)

            if not modes:
                return []

            # Thin by minimum fractional‑octave spacing
            ratio_threshold = 2 ** min_spacing_octaves
            kept = []
            for m in modes:
                if not kept or (m / kept[-1]) >= ratio_threshold:
                    kept.append(m)
                if len(kept) >= max_modes:
                    break

            logging.info(f"Selected {len(kept)} spaced modes (≤{max_modes}) in 20–300 Hz")
            return kept

        except Exception as e:
            logging.error(f"Error calculating room modes: {e}")
            return []


def main():
    logging.info("Sonara Python audio analyzer starting")
    logging.info(f"Command line arguments: {sys.argv}")

    if len(sys.argv) < 3:
        error_msg = "Usage: python analyze_audio.py <recorded_file> <signal_id> [output_file] [room_data_json]"
        logging.error(error_msg)
        print(json.dumps({"error": error_msg}))
        sys.exit(1)

    recorded_file = sys.argv[1]
    signal_id = sys.argv[2]
    output_file = sys.argv[3] if len(sys.argv) > 3 else None
    room_data_json = sys.argv[4] if len(sys.argv) > 4 else None

    logging.info(f"Input file: {recorded_file}")
    logging.info(f"Signal ID: {signal_id}")
    if output_file:
        logging.info(f"Output file: {output_file}")
    else:
        logging.warning("No output file specified, will use stdout (deprecated)")
    if room_data_json:
        logging.info(f"Room data provided: {room_data_json}")
    else:
        logging.info("No room data provided, using basic analysis")

    # Parse room data JSON if provided
    room_data = None
    if room_data_json:
        try:
            room_data = json.loads(room_data_json)
            logging.info(f"Parsed room data: {room_data}")
        except json.JSONDecodeError as e:
            logging.error(f"Failed to parse room data JSON: {e}")
            room_data = None

    # Use new FrequencyAnalyzer for sweep deconvolution
    try:
        logging.info("Initializing FrequencyAnalyzer for sweep deconvolution")
        analyzer = FrequencyAnalyzer()
        logging.info("Starting sweep deconvolution analysis")
        frequencies, response_db, room_modes = analyzer.analyze_sweep_deconvolution(
            recorded_file, signal_id, room_data
        )
        logging.info(f"Sweep deconvolution completed - generated {len(frequencies)} frequency points, {len(room_modes)} room modes")

        # Format for frontend with log-spaced resampling
        logging.info(f"Resampling from {len(frequencies)} to 300 log-spaced points")
        display_freqs, display_mags = analyzer.resample_log_spaced(
            frequencies, response_db, num_points=300
        )
        frequency_data = [
            {"frequency": float(f), "magnitude": float(m)}
            for f, m in zip(display_freqs, display_mags)
        ]
        logging.info(f"Resampled to {len(frequency_data)} log-spaced frequency points")

        result = {
            "frequency_data": frequency_data,
            "analysis_type": "sweep_deconvolution",
            "smoothing": "1/12 octave",
            "fft_size": analyzer.fft_size,
            "reference": signal_id,
            "rt60": 0.5,  # Placeholder - can be calculated from impulse response
            "room_modes": room_modes  # Calculated room mode frequencies
        }
        logging.info("Sweep deconvolution result prepared successfully")

    except Exception as e:
        error_msg = f"Sweep deconvolution failed: {str(e)}"
        logging.error(error_msg)
        result = {"error": error_msg}

    # Output results to file or stdout
    if output_file:
        logging.info(f"Writing results to file: {output_file}")
        try:
            with open(output_file, 'w') as f:
                f.write(json.dumps(result))
            logging.info("Results written to file successfully")
        except Exception as e:
            logging.error(f"Failed to write results to file {output_file}: {e}")
            print(json.dumps({"error": f"Failed to write output file: {str(e)}"}))
            sys.exit(1)
    else:
        # Backward compatibility fallback
        logging.warning("Writing results to stdout (deprecated)")
        print(json.dumps(result))
        logging.info("JSON output sent to stdout")

    if "error" in result:
        logging.error("Exiting with error status")
        sys.exit(1)

    logging.info("Python script completed successfully")


if __name__ == "__main__":
    main()
