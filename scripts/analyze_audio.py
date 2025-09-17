#!/usr/bin/env python3
"""
Sonara Audio Analyzer
Performs FFT analysis on audio files with FIFINE K669 microphone calibration
"""

import sys
import json
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

    def analyze(self, filepath):
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

            # Detect room modes
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
    if len(sys.argv) != 2:
        print(json.dumps({"error": "Usage: python analyze_audio.py <audio_file>"}))
        sys.exit(1)

    analyzer = AudioAnalyzer()
    result = analyzer.analyze(sys.argv[1])
    print(json.dumps(result))

    if "error" in result:
        sys.exit(1)


if __name__ == "__main__":
    main()
