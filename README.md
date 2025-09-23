# Sonara, 'See your sound clearly'

Sonara is a consumer application for analyzing your room's frequency response and echo characteristics using a USB microphone. It provides actionable insights for optimizing speaker placement and sound treatment to achieve more accurate, professional-quality audio in your listening space.

The application features a modern web interface built with React and TypeScript, powered by a Go backend that orchestrates Python-based audio signal processing with libraries like NumPy, SciPy, and Librosa. Data is stored in PostgreSQL with file assets on AWS S3, while OpenAI integration delivers intelligent recommendations for acoustic improvements.