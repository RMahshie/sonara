# Sonara, 'See your sound clearly'

Sonara is a consumer application for analyzing your room's frequency response and echo characteristics using a USB microphone. It provides actionable insights for optimizing speaker placement and sound treatment to achieve more accurate, professional-quality audio in your listening space.

The application features a modern web interface built with React and TypeScript, powered by a Go backend that orchestrates Python-based audio signal processing with libraries like NumPy, SciPy, and Librosa. Data is stored in PostgreSQL with file assets on AWS S3, while OpenAI integration delivers intelligent recommendations for acoustic improvements.

![Sonara Start Screen Screenshot](photos/Screenshot%202025-09-30%20at%2011.21.41%E2%80%AFPM.png)
*The main interface featuring a start button that initiates live room analysis by playing test signals while recording simultaneously.*

![Sonara Application Screenshot](photos/Screenshot%202025-09-30%20at%2011.17.40%E2%80%AFPM.png)
*Interactive frequency response graph displaying room acoustics analysis with detailed measurements.*


## ⚙️ Environment Configuration

**PYTHON_CMD** determines how the Go backend calls the Python audio analyzer:

- **Development**: `export ENVIRONMENT=dev`
- **Testing**: `export ENVIRONMENT=test`
- **Production**: `export ENVIRONMENT=prod`

This allows the same codebase to work with containerized Python in development and direct execution in production.