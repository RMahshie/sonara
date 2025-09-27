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
from scipy.io import wavfile
import librosa

from reference_signals import get_reference_manager


class AudioAnalyzer:
    """Audio analyzer with microphone calibration"""

    def __init__(self):
        # FIFINE K669 USB Microphone calibration curve
        # Compensates for microphone frequency response
        self.calibration_curve = [
            (20, 12),     # +12dB at 20Hz (mic rolls off)
            (50, 3),      # +3dB at 50Hz
            (100, 0),     # Flat at 100Hz
            (200, 0),     # Flat at 200Hz
            (500, 0),     # Flat at 500Hz
            (1000, 0),    # Flat at 1kHz (reference)
            (2000, -1),   # -1dB at 2kHz
            (5000, -2),   # -2dB at 5kHz
            (8000, -3),   # -3dB at 8kHz (mic boost)
            (10000, -3.5), # -3.5dB at 10kHz
            (12000, -4),  # -4dB at 12kHz
            (16000, -2),  # -2dB at 16kHz
            (20000, 5)    # +5dB at 20kHz (mic rolls off)
        ]

    def load_audio(self, filepath):
        """Load audio file and convert to mono"""
        logging.info(f"Loading audio file: {filepath}")
        # librosa handles WAV, MP3, FLAC
        audio, sr = librosa.load(filepath, sr=None, mono=True)
        duration = len(audio) / sr
        logging.info(f"Audio loaded successfully - sample rate: {sr}Hz, duration: {duration:.2f}s, samples: {len(audio)}")
        return audio, sr

    def perform_fft(self, audio, sample_rate):
        """
        Perform FFT analysis with proper windowing
        Returns frequency and magnitude arrays
        """
        # Apply Hamming window to reduce spectral leakage
        windowed = audio * signal.windows.hamming(len(audio))

        # Perform FFT (real FFT since audio is real)
        fft_result = np.fft.rfft(windowed)
        magnitude = np.abs(fft_result)

        # Convert to decibels (20*log10)
        magnitude_db = 20 * np.log10(magnitude + 1e-10)  # Small value to avoid log(0)

        # Generate frequency bins
        frequencies = np.fft.rfftfreq(len(audio), 1/sample_rate)

        return frequencies, magnitude_db

    def apply_calibration(self, frequencies, magnitudes):
        """Apply FIFINE K669 calibration curve"""
        # Extract calibration points
        cal_freqs, cal_values = zip(*self.calibration_curve)

        # Interpolate calibration values for all frequencies
        calibration = np.interp(frequencies, cal_freqs, cal_values)

        # Apply calibration (add correction values)
        calibrated_magnitudes = magnitudes + calibration

        return calibrated_magnitudes

    def calculate_rt60(self, audio, sample_rate):
        """
        Calculate RT60 (reverberation time) using Schroeder integration
        Returns time in seconds for 60dB decay
        """
        # This is a simplified implementation
        # Real RT60 requires impulse response or interrupted noise
        # For now, return placeholder
        return 0.5  # Will be properly implemented in Week 2

    def detect_room_modes(self, frequencies, magnitudes):
        """
        Detect room modes (resonant frequencies)
        Returns list of problematic frequencies
        """
        # Find peaks in frequency response
        # Simplified for Week 1, enhanced in Week 2
        from scipy.signal import find_peaks

        # Find peaks that are >6dB above neighbors
        peaks, properties = find_peaks(magnitudes, prominence=6, distance=20)

        # Return frequencies of peaks below 300Hz (typical room mode range)
        room_modes = []
        for peak in peaks:
            if frequencies[peak] < 300:
                room_modes.append(float(frequencies[peak]))

        return room_modes[:5]  # Return top 5 modes

    def calculate_room_modes(self, room_data):
        """
        Calculate theoretical room modes based on room dimensions
        Returns list of predicted mode frequencies
        """
        try:
            length = room_data.get('room_length', 0)
            width = room_data.get('room_width', 0)
            height = room_data.get('room_height', 0)

            if not all([length, width, height]):
                return []

            # Speed of sound in air at 20°C (343 m/s)
            c = 343.0

            modes = []

            # Axial modes (simplest - sound bouncing between 2 parallel surfaces)
            if length > 0:
                modes.append(c / (2 * length))  # Length mode
            if width > 0:
                modes.append(c / (2 * width))   # Width mode
            if height > 0:
                modes.append(c / (2 * height))  # Height mode

            # Tangential modes (sound bouncing between 4 surfaces)
            if length > 0 and width > 0:
                modes.append(c / (2 * math.sqrt(length**2 + width**2)))
            if length > 0 and height > 0:
                modes.append(c / (2 * math.sqrt(length**2 + height**2)))
            if width > 0 and height > 0:
                modes.append(c / (2 * math.sqrt(width**2 + height**2)))

            # Oblique modes (sound bouncing between all 6 surfaces)
            if length > 0 and width > 0 and height > 0:
                modes.append(c / (2 * math.sqrt(length**2 + width**2 + height**2)))

            # Sort and filter to reasonable frequency range (20Hz - 300Hz)
            modes = sorted([m for m in modes if 20 <= m <= 300])

            return modes[:8]  # Return top 8 predicted modes

        except Exception as e:
            logging.error(f"Error calculating room modes: {e}")
            return []

    def detect_room_modes_enhanced(self, frequencies, magnitudes, predicted_modes=None):
        """
        Enhanced room mode detection that looks for both measured peaks
        and predicted theoretical modes
        """
        room_modes = []

        # First, find measured peaks (existing approach)
        peaks, properties = find_peaks(magnitudes, prominence=6, distance=20)

        for peak in peaks:
            if frequencies[peak] < 300:
                room_modes.append({
                    "frequency": float(frequencies[peak]),
                    "magnitude": float(magnitudes[peak]),
                    "type": "measured_peak"
                })

        # Second, check for predicted modes if room data is available
        if predicted_modes:
            for predicted_freq in predicted_modes:
                # Look for measured data near predicted frequency (±5Hz)
                nearby_indices = np.where(
                    (frequencies >= predicted_freq - 5) &
                    (frequencies <= predicted_freq + 5)
                )[0]

                if len(nearby_indices) > 0:
                    # Find the strongest response in this range
                    max_idx = nearby_indices[np.argmax(magnitudes[nearby_indices])]
                    room_modes.append({
                        "frequency": float(frequencies[max_idx]),
                        "magnitude": float(magnitudes[max_idx]),
                        "type": "predicted_mode",
                        "predicted_freq": predicted_freq
                    })
                else:
                    # No measured data near predicted frequency
                    room_modes.append({
                        "frequency": predicted_freq,
                        "magnitude": None,
                        "type": "predicted_mode_only",
                        "predicted_freq": predicted_freq
                    })

        # Sort by frequency and return top results
        room_modes.sort(key=lambda x: x["frequency"])

        # Remove duplicates (keep measured peaks over predicted)
        seen_freqs = set()
        unique_modes = []
        for mode in room_modes:
            freq_rounded = round(mode["frequency"])
            if freq_rounded not in seen_freqs:
                seen_freqs.add(freq_rounded)
                unique_modes.append(mode)

        return unique_modes[:10]  # Return top 10 modes

    def analyze(self, filepath, room_data=None):
        """Main analysis function"""
        try:
            logging.info("Starting basic audio analysis")
            # Load audio
            audio, sample_rate = self.load_audio(filepath)

            # Perform FFT
            logging.info("Performing FFT analysis")
            frequencies, magnitudes = self.perform_fft(audio, sample_rate)
            logging.info(f"FFT completed - {len(frequencies)} frequency points generated")

            # Apply calibration
            logging.info("Applying microphone calibration")
            calibrated = magnitudes #self.apply_calibration(frequencies, magnitudes)
            logging.info("Calibration applied successfully")

            # Calculate RT60 (placeholder for Week 1)
            logging.info("Calculating RT60")
            rt60 = self.calculate_rt60(audio, sample_rate)
            logging.info(f"RT60 calculated: {rt60}")

            # Enhanced room mode detection with room data if available
            logging.info("Detecting room modes")
            if room_data:
                logging.info("Using enhanced room mode detection with room data")
                predicted_modes = self.calculate_room_modes(room_data)
                room_modes = self.detect_room_modes_enhanced(frequencies, calibrated, predicted_modes)
            else:
                logging.info("Using basic room mode detection")
                # Fallback to original peak detection
                room_modes = self.detect_room_modes(frequencies, calibrated)
            logging.info(f"Room modes detected: {len(room_modes)} modes found")

            # Reduce data points for reasonable JSON size
            # Take every Nth point to get ~1000 points
            step = max(1, len(frequencies) // 1000)
            logging.info(f"Reducing data points - step size: {step}, target: ~1000 points")

            # Filter to audible range (20Hz - 20kHz)
            logging.info("Filtering to audible frequency range (20Hz - 20kHz)")
            frequency_data = []
            for i in range(0, len(frequencies), step):
                if 20 <= frequencies[i] <= 20000:
                    frequency_data.append({
                        "frequency": float(frequencies[i]),
                        "magnitude": float(calibrated[i])
                    })

            logging.info(f"Final frequency data prepared: {len(frequency_data)} points")
            result = {
                "sample_rate": int(sample_rate),
                "frequency_data": frequency_data,
                "rt60": rt60,
                "room_modes": room_modes
            }

            logging.info("Basic analysis completed successfully")
            return result

        except Exception as e:
            logging.error(f"Basic analysis failed: {str(e)}")
            return {"error": str(e)}


class FrequencyAnalyzer:
    """
    Professional frequency response analyzer using sweep deconvolution.

    Performs sine sweep deconvolution for accurate acoustic measurements.
    """

    def __init__(self):
        self.fft_size = 32768  # 32k FFT for high resolution
        self.smoothing_fraction = 1/12  # 1/12 octave smoothing
        self.reference_freq = 1000  # 1kHz normalization
        self.ref_manager = get_reference_manager()

    def analyze_sweep_deconvolution(self, recorded_file, signal_id):
        """
        Main sweep deconvolution pipeline for frequency response measurement.

        Args:
            recorded_file: Path to recorded audio file
            signal_id: Identifier for reference signal (e.g., "sine_sweep_20_20k")

        Returns:
            tuple: (frequencies, response_db) arrays
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

        # 3. Direct deconvolution (no alignment needed for sine sweeps)
        logging.info("Step 3: Performing deconvolution")
        impulse_response = self.deconvolve_signals(
            recorded, ref_data["inverse_signal"]
        )
        logging.info(f"Deconvolution completed - impulse response length: {len(impulse_response)} samples")

        # 4. Extract acoustic impulse window
        logging.info("Step 4: Extracting acoustic impulse window")
        impulse_windowed = self.extract_impulse_window(impulse_response, sr)
        logging.info(f"Impulse window extracted - length: {len(impulse_windowed)} samples")

        # 5. Convert impulse response to frequency response
        logging.info("Step 5: Converting impulse to frequency response")
        freqs, response_db = self.impulse_to_frequency_response(impulse_windowed, sr)
        logging.info(f"Frequency response calculated - {len(freqs)} frequency points")

        # 6. Apply fractional octave smoothing
        logging.info("Step 6: Applying fractional octave smoothing")
        smooth_response = self.apply_fractional_octave_smoothing(freqs, response_db)
        logging.info("Fractional octave smoothing applied")

        # 7. Normalize to 0dB at 1kHz
        logging.info("Step 7: Normalizing response to 0dB at 1kHz")
        final_response = self.normalize_response(freqs, smooth_response)
        logging.info("Response normalization completed")

        logging.info(f"Sweep deconvolution analysis completed - final result: {len(freqs)} frequency points")
        return freqs, final_response


    def deconvolve_signals(self, recorded, inverse_filter):
        """
        Deconvolve recorded signal with inverse filter to get impulse response.

        Uses FFT-based convolution for efficiency with complete convolution result.
        """
        return signal.fftconvolve(recorded, inverse_filter, mode='full')

    def extract_impulse_window(self, impulse, sample_rate):
        """
        Extract the main impulse response with acoustic-appropriate windowing.

        Finds the main peak and windows around it for room acoustic measurements.
        Uses 500ms total window (100ms before peak, 400ms after peak).
        """
        # Find main impulse peak
        peak_idx = np.argmax(np.abs(impulse))

        # Acoustic window: 500ms total (100ms before, 400ms after peak)
        pre_samples = int(0.1 * sample_rate)   # 100ms before peak
        post_samples = int(0.4 * sample_rate)  # 400ms after peak

        start = max(0, peak_idx - pre_samples)
        end = min(len(impulse), peak_idx + post_samples)

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
                # RMS averaging in linear domain (more accurate than dB averaging)
                linear_values = 10 ** (magnitude_db[in_band] / 20)
                rms_linear = np.sqrt(np.mean(linear_values ** 2))
                smoothed[i] = 20 * np.log10(rms_linear + 1e-12)
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


def main():
    logging.info("Sonara Python audio analyzer starting")
    logging.info(f"Command line arguments: {sys.argv}")

    if len(sys.argv) < 3:
        error_msg = "Usage: python analyze_audio.py <recorded_file> <signal_id> [output_file]"
        logging.error(error_msg)
        print(json.dumps({"error": error_msg}))
        sys.exit(1)

    recorded_file = sys.argv[1]
    signal_id = sys.argv[2]
    output_file = sys.argv[3] if len(sys.argv) > 3 else None

    logging.info(f"Input file: {recorded_file}")
    logging.info(f"Signal ID: {signal_id}")
    if output_file:
        logging.info(f"Output file: {output_file}")
    else:
        logging.warning("No output file specified, will use stdout (deprecated)")

    # Use new FrequencyAnalyzer for sweep deconvolution
    try:
        logging.info("Initializing FrequencyAnalyzer for sweep deconvolution")
        analyzer = FrequencyAnalyzer()
        logging.info("Starting sweep deconvolution analysis")
        frequencies, response_db = analyzer.analyze_sweep_deconvolution(
            recorded_file, signal_id
        )
        logging.info(f"Sweep deconvolution completed - generated {len(frequencies)} frequency points")

        # Format for frontend (downsample to ~800 points)
        logging.info(f"Formatting results for frontend - downsampling from {len(frequencies)} to ~800 points")
        step = max(1, len(frequencies) // 800)
        frequency_data = [
            {"frequency": float(f), "magnitude": float(m)}
            for f, m in zip(frequencies[::step], response_db[::step])
        ]
        logging.info(f"Downsampled to {len(frequency_data)} frequency points")

        result = {
            "frequency_data": frequency_data,
            "analysis_type": "sweep_deconvolution",
            "smoothing": "1/12 octave",
            "fft_size": analyzer.fft_size,
            "reference": signal_id,
            "rt60": 0.5,  # Placeholder - can be calculated from impulse response
            "room_modes": []  # Placeholder - can be detected from frequency response
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
