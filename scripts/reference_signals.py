#!/usr/bin/env python3
"""
Reference Signal Manager for Sonara Acoustic Analysis

Manages reference test signals and their pre-computed inverse filters
for frequency response measurement via sweep deconvolution.
"""

import os
import logging
import numpy as np
import librosa


class ReferenceSignalManager:
    """
    Manages reference signals for acoustic measurement.

    Stores original test signals and their inverse filters for deconvolution.
    Pre-computes FFTs for performance.
    """

    def __init__(self):
        self.base_path = os.path.join(os.path.dirname(__file__), 'reference_signals')
        self.signals = {
            "sine_sweep_20_20k": {
                "sweep": "sweep-20-20k-10s.wav",
                "inverse": "sweep-20-20k-inverse.wav",
                "description": "10s logarithmic sine sweep 20Hz-20kHz"
            }
        }
        self.fft_cache = {}
        self._load_cache()

    def _load_cache(self):
        """Pre-compute and cache FFTs for performance"""
        for signal_id, config in self.signals.items():
            sweep_path = os.path.join(self.base_path, config["sweep"])
            inverse_path = os.path.join(self.base_path, config["inverse"])

            try:
                # Load signals
                sweep, sr = librosa.load(sweep_path, sr=None, mono=True)
                inverse, _ = librosa.load(inverse_path, sr=None, mono=True)

                # Cache FFTs (32k points for consistency)
                self.fft_cache[signal_id] = {
                    "sweep_fft": np.fft.rfft(sweep, n=32768),
                    "inverse_fft": np.fft.rfft(inverse, n=32768),
                    "sweep_signal": sweep,
                    "inverse_signal": inverse,
                    "sample_rate": sr
                }
                logging.info(f"Loaded reference signal: {signal_id}")

            except Exception as e:
                logging.error(f"Error loading reference signal {signal_id}: {e}")
                self.fft_cache[signal_id] = None

    def get_signal_data(self, signal_id):
        """Get cached signal data for analysis"""
        return self.fft_cache.get(signal_id)

    def get_sweep_path(self, signal_id):
        """Get filesystem path for frontend playback"""
        config = self.signals.get(signal_id)
        if config:
            return f"/test-signals/{config['sweep']}"
        return None

    def list_available_signals(self):
        """List all available reference signals"""
        return list(self.signals.keys())

    def validate_signal(self, signal_id):
        """Validate that a signal is properly loaded"""
        data = self.get_signal_data(signal_id)
        if data is None:
            return False

        required_keys = ["sweep_fft", "inverse_fft", "sweep_signal", "inverse_signal", "sample_rate"]
        return all(key in data for key in required_keys)


# Global instance for easy access
_ref_manager = None

def get_reference_manager():
    """Get global reference signal manager instance"""
    global _ref_manager
    if _ref_manager is None:
        _ref_manager = ReferenceSignalManager()
    return _ref_manager
