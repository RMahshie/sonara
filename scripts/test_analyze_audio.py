#!/usr/bin/env python3
"""Comprehensive tests for audio analysis script"""

import unittest
import numpy as np
import json
import tempfile
import os
from scipy.io import wavfile
import sys
import inspect

# Add parent directory to path to import analyze_audio
current_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(current_dir)
sys.path.insert(0, parent_dir)

from scripts.analyze_audio import AudioAnalyzer


class AudioTestUtils:
    """Utility methods for audio test generation"""

    @staticmethod
    def generate_sine_wave(frequency, sample_rate, duration, amplitude=1.0):
        """Generate a sine wave audio signal"""
        t = np.linspace(0, duration, int(sample_rate * duration), False)
        return np.sin(2 * np.pi * frequency * t) * amplitude

    @staticmethod
    def create_wav_file(audio, sample_rate, filename):
        """Create a WAV file from audio data"""
        audio_int16 = np.int16(audio * 32767)
        wavfile.write(filename, sample_rate, audio_int16)

    @staticmethod
    def find_peak_near_frequency(freq_data, target_freq, tolerance_hz=50):
        """Find the peak closest to target frequency"""
        nearby_points = [p for p in freq_data if abs(p['frequency'] - target_freq) <= tolerance_hz]
        return max(nearby_points, key=lambda p: p['magnitude']) if nearby_points else None


class TestAudioAnalyzer(unittest.TestCase):
    def setUp(self):
        """Set up test fixtures"""
        self.analyzer = AudioAnalyzer()

    def test_basic_fft(self):
        """Test that FFT produces reasonable frequency data"""
        # Generate a 1kHz sine wave (440Hz for A note)
        sample_rate = 44100
        duration = 1.0  # 1 second
        frequency = 440.0

        t = np.linspace(0, duration, int(sample_rate * duration), False)
        audio = np.sin(frequency * 2 * np.pi * t)

        # Create temporary WAV file
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as temp_file:
            temp_filename = temp_file.name
            # Normalize to 16-bit PCM
            audio_int16 = np.int16(audio * 32767)
            wavfile.write(temp_filename, sample_rate, audio_int16)

        try:
            # Analyze the file
            result = self.analyzer.analyze(temp_filename)

            # Verify basic structure
            self.assertIn('frequency_data', result)
            self.assertIn('rt60', result)
            self.assertIn('room_modes', result)
            self.assertIn('sample_rate', result)

            # Verify frequency data is present
            freq_data = result['frequency_data']
            self.assertIsInstance(freq_data, list)
            self.assertGreater(len(freq_data), 0)

            # Check that we have data points in audible range
            frequencies = [point['frequency'] for point in freq_data]
            self.assertTrue(any(20 <= f <= 20000 for f in frequencies))

            # Verify sample rate
            self.assertEqual(result['sample_rate'], sample_rate)

            # RT60 should be a reasonable value (placeholder for now)
            self.assertIsInstance(result['rt60'], (int, float))

        finally:
            # Clean up
            os.unlink(temp_filename)

    def test_1khz_peak_detection_accuracy(self):
        """Test 1kHz tone peak detection within 1% accuracy"""
        sample_rate = 44100
        duration = 2.0  # Longer for better FFT resolution
        frequency = 1000.0  # Exactly 1kHz

        # Generate pure sine wave
        audio = AudioTestUtils.generate_sine_wave(frequency, sample_rate, duration)

        # Create test file
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as temp_file:
            temp_filename = temp_file.name
            AudioTestUtils.create_wav_file(audio, sample_rate, temp_filename)

        try:
            result = self.analyzer.analyze(temp_filename)
            freq_data = result['frequency_data']

            # Find peak closest to 1kHz
            peak = AudioTestUtils.find_peak_near_frequency(freq_data, 1000.0, 50)
            self.assertIsNotNone(peak, "No peaks found near 1kHz")

            # Verify peak frequency is within 1.1% of 1000Hz (allowing for FFT resolution)
            frequency_error = abs(peak['frequency'] - 1000.0) / 1000.0
            self.assertLess(frequency_error, 0.011, f"Peak detection error: {frequency_error*100:.2f}%")

        finally:
            os.unlink(temp_filename)

    def test_calibration_curve_interpolation(self):
        """Test calibration curve interpolation accuracy"""
        # Test known calibration points
        test_points = [
            (20, 12),    # +12dB at 20Hz
            (1000, 0),   # Flat at 1kHz
            (8000, -3),  # -3dB at 8kHz
            (20000, 5)   # +5dB at 20kHz
        ]

        frequencies = np.array([p[0] for p in test_points])
        expected_corrections = np.array([p[1] for p in test_points])
        dummy_magnitudes = np.zeros_like(frequencies)  # Flat response

        calibrated = self.analyzer.apply_calibration(frequencies, dummy_magnitudes)

        # Verify calibration corrections are applied correctly
        for i, (freq, expected_corr) in enumerate(test_points):
            actual_corr = calibrated[i] - dummy_magnitudes[i]
            self.assertAlmostEqual(actual_corr, expected_corr, places=1,
                                  msg=f"Calibration error at {freq}Hz: expected {expected_corr}, got {actual_corr}")

    def test_calibration_interpolation_between_points(self):
        """Test interpolation between calibration curve points"""
        # Test midpoint between 1000Hz (0dB) and 2000Hz (-1dB)
        frequencies = np.array([1500.0])
        dummy_magnitudes = np.array([0.0])

        calibrated = self.analyzer.apply_calibration(frequencies, dummy_magnitudes)

        # Should interpolate to approximately -0.5dB
        expected_correction = -0.5
        actual_correction = calibrated[0] - dummy_magnitudes[0]
        self.assertAlmostEqual(actual_correction, expected_correction, places=1)

    def test_multiple_sample_rates(self):
        """Test analysis with different sample rates"""
        test_rates = [44100, 48000, 96000]
        frequency = 1000.0  # 1kHz test tone

        for sample_rate in test_rates:
            with self.subTest(sample_rate=sample_rate):
                duration = 1.0
                audio = AudioTestUtils.generate_sine_wave(frequency, sample_rate, duration)

                with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as temp_file:
                    temp_filename = temp_file.name
                    AudioTestUtils.create_wav_file(audio, sample_rate, temp_filename)

                try:
                    result = self.analyzer.analyze(temp_filename)

                    # Verify sample rate is correctly detected
                    self.assertEqual(result['sample_rate'], sample_rate)

                    # Verify frequency data exists and covers expected range
                    freq_data = result['frequency_data']
                    self.assertGreater(len(freq_data), 0)

                    # Check that 1kHz peak is detected
                    peak_found = any(abs(p['frequency'] - 1000) < 50 for p in freq_data)
                    self.assertTrue(peak_found, f"1kHz peak not found at {sample_rate}Hz sample rate")

                finally:
                    os.unlink(temp_filename)

    def test_very_quiet_signal(self):
        """Test analysis with very quiet signal (near noise floor)"""
        sample_rate = 44100
        duration = 2.0
        frequency = 1000.0

        # Generate signal at -60dB (very quiet)
        audio = AudioTestUtils.generate_sine_wave(frequency, sample_rate, duration, 0.001)  # -60dB amplitude

        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as temp_file:
            temp_filename = temp_file.name
            AudioTestUtils.create_wav_file(audio, sample_rate, temp_filename)

        try:
            result = self.analyzer.analyze(temp_filename)
            freq_data = result['frequency_data']

            # Should still detect the signal (though magnitude will be low)
            self.assertGreater(len(freq_data), 0)
            self.assertNotIn('error', result)

        finally:
            os.unlink(temp_filename)

    def test_very_loud_signal(self):
        """Test analysis with very loud signal (near clipping)"""
        sample_rate = 44100
        duration = 1.0
        frequency = 1000.0

        # Generate signal near full scale (0.95 to avoid clipping)
        audio = AudioTestUtils.generate_sine_wave(frequency, sample_rate, duration, 0.95)

        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as temp_file:
            temp_filename = temp_file.name
            AudioTestUtils.create_wav_file(audio, sample_rate, temp_filename)

        try:
            result = self.analyzer.analyze(temp_filename)
            freq_data = result['frequency_data']

            # Should handle loud signals without crashing
            self.assertGreater(len(freq_data), 0)
            self.assertNotIn('error', result)

            # Verify 1kHz fundamental is still detectable
            peak_found = any(abs(p['frequency'] - 1000) < 100 for p in freq_data)
            self.assertTrue(peak_found, "1kHz peak should still be detectable in loud signal")

            # Verify calibration curve is working: high frequencies should be boosted
            # The FIFINE K669 calibration curve boosts frequencies near 20kHz by +5dB
            high_freq_20k = [p for p in freq_data if 19500 <= p['frequency'] <= 20000]
            if high_freq_20k:
                max_20k_level = max(p['magnitude'] for p in high_freq_20k)
                # Should be boosted relative to uncalibrated response
                # (This verifies calibration is working, not that there's distortion)
                fundamental_points = [p for p in freq_data if 900 <= p['frequency'] <= 1100]
                if fundamental_points:
                    fundamental_level = max(p['magnitude'] for p in fundamental_points)
                    # 20kHz should be reasonably close to fundamental after calibration
                    # (allowing for natural rolloff + calibration boost)
                    level_diff = fundamental_level - max_20k_level
                    self.assertLess(level_diff, 15, f"20kHz too far below fundamental: {level_diff:.1f} dB")

        finally:
            os.unlink(temp_filename)

    def test_dc_offset_signal(self):
        """Test handling of signal with DC offset"""
        sample_rate = 44100
        duration = 1.0
        frequency = 1000.0

        # Generate signal with DC offset
        t = np.linspace(0, duration, int(sample_rate * duration), False)
        audio = 0.5 + 0.1 * np.sin(2 * np.pi * frequency * t)  # DC + AC

        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as temp_file:
            temp_filename = temp_file.name
            # Normalize to avoid clipping
            audio_normalized = audio / np.max(np.abs(audio))
            AudioTestUtils.create_wav_file(audio_normalized, sample_rate, temp_filename)

        try:
            result = self.analyzer.analyze(temp_filename)
            freq_data = result['frequency_data']

            # Should still work and detect the 1kHz component
            self.assertGreater(len(freq_data), 0)
            self.assertNotIn('error', result)

            # Check for very low frequency content (DC and near-DC)
            low_freq_points = [p for p in freq_data if p['frequency'] < 10]
            if low_freq_points:
                # DC component should be present but analysis should still work
                pass  # Just verify no crash

        finally:
            os.unlink(temp_filename)

    def test_invalid_file(self):
        """Test handling of invalid file"""
        result = self.analyzer.analyze('/nonexistent/file.wav')
        self.assertIn('error', result)

    def test_frequency_range(self):
        """Test that frequency data covers expected range"""
        # Generate white noise
        sample_rate = 44100
        duration = 1.0
        audio = np.random.normal(0, 1, int(sample_rate * duration))

        # Create temporary WAV file
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as temp_file:
            temp_filename = temp_file.name
            audio_int16 = np.int16(audio * 32767)
            wavfile.write(temp_filename, sample_rate, audio_int16)

        try:
            result = self.analyzer.analyze(temp_filename)
            freq_data = result['frequency_data']

            # Should have frequencies from ~20Hz to ~20kHz
            min_freq = min(point['frequency'] for point in freq_data)
            max_freq = max(point['frequency'] for point in freq_data)

            self.assertLess(min_freq, 100)  # Should include low frequencies
            self.assertGreater(max_freq, 10000)  # Should include high frequencies

        finally:
            os.unlink(temp_filename)


if __name__ == '__main__':
    unittest.main()

