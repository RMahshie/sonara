#!/usr/bin/env python3
"""
Sonara Audio Analyzer
Performs FFT analysis on audio files with FIFINE K669 microphone calibration
"""

import sys
import json
import math
import numpy as np
from scipy import signal
from scipy.io import wavfile
import librosa


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
        # librosa handles WAV, MP3, FLAC
        audio, sr = librosa.load(filepath, sr=None, mono=True)
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
            print(f"Error calculating room modes: {e}")
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
            # Load audio
            audio, sample_rate = self.load_audio(filepath)

            # Perform FFT
            frequencies, magnitudes = self.perform_fft(audio, sample_rate)

            # Apply calibration
            calibrated = self.apply_calibration(frequencies, magnitudes)

            # Calculate RT60 (placeholder for Week 1)
            rt60 = self.calculate_rt60(audio, sample_rate)

            # Enhanced room mode detection with room data if available
            if room_data:
                predicted_modes = self.calculate_room_modes(room_data)
                room_modes = self.detect_room_modes_enhanced(frequencies, calibrated, predicted_modes)
            else:
                # Fallback to original peak detection
                room_modes = self.detect_room_modes(frequencies, calibrated)

            # Reduce data points for reasonable JSON size
            # Take every Nth point to get ~1000 points
            step = max(1, len(frequencies) // 1000)

            # Filter to audible range (20Hz - 20kHz)
            frequency_data = []
            for i in range(0, len(frequencies), step):
                if 20 <= frequencies[i] <= 20000:
                    frequency_data.append({
                        "frequency": float(frequencies[i]),
                        "magnitude": float(calibrated[i])
                    })

            result = {
                "sample_rate": int(sample_rate),
                "frequency_data": frequency_data,
                "rt60": rt60,
                "room_modes": room_modes
            }

            return result

        except Exception as e:
            return {"error": str(e)}


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: python analyze_audio.py <audio_file> [room_data_json]"}))
        sys.exit(1)

    analyzer = AudioAnalyzer()

    # Check if room data is provided
    if len(sys.argv) >= 3:
        try:
            room_data = json.loads(sys.argv[2])
            result = analyzer.analyze(sys.argv[1], room_data)
        except json.JSONDecodeError as e:
            result = {"error": f"Invalid room data JSON: {e}"}
    else:
        # No room data provided, use original approach
        result = analyzer.analyze(sys.argv[1])

    print(json.dumps(result))

    if "error" in result:
        sys.exit(1)


if __name__ == "__main__":
    main()
