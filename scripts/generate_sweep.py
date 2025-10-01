import numpy as np
from scipy.io import wavfile
import os

def generate_exponential_sweep(duration=10, sample_rate=44100):
    """
    Generate exponential sine sweep for acoustic measurement.

    Exponential sweeps provide equal energy per octave, which matches
    how humans perceive sound and gives better SNR across the frequency range.
    """
    t = np.linspace(0, duration, int(sample_rate * duration))
    f0, f1 = 20, 20000

    # Exponential sweep formula
    R = np.log(f1 / f0)
    K = duration * f0 / R
    L = duration / R
    sweep = np.sin(2 * np.pi * K * (np.exp(t / L) - 1))

    # Scale to prevent clipping (0.5 = -6dBFS)
    sweep = sweep * 0.5
    return sweep

# Generate 10-second exponential sweep at 44.1kHz
duration = 10
sample_rate = 44100
sweep = generate_exponential_sweep(duration, sample_rate)

output_dir = os.path.join(os.path.dirname(__file__), 'reference_signals')
os.makedirs(output_dir, exist_ok=True)

# Save only the sweep file (no inverse needed for spectral division)
wavfile.write(os.path.join(output_dir, 'exp-sweep-44.wav'), sample_rate, (sweep * 32767).astype(np.int16))

print(f"Generated exponential sweep file at {sample_rate}Hz")
print(f"Duration: {duration}s, Frequency range: 20Hz - 20kHz")
print("File: exp-sweep-44.wav")