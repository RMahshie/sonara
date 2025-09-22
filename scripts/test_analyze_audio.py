#!/usr/bin/env python3
"""Tests for audio analysis script"""

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

