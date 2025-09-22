import { useParams } from 'react-router-dom'
import FrequencyChart from './FrequencyChart'

const Results = () => {
  const { id } = useParams<{ id: string }>()

  // Mock data - this will come from the API
  const mockData = {
    filename: 'sample_audio.wav',
    duration: '3:24',
    sampleRate: 44100,
    frequencies: [
      { frequency: 100, amplitude: 0.8 },
      { frequency: 200, amplitude: 0.6 },
      { frequency: 500, amplitude: 0.9 },
      { frequency: 1000, amplitude: 0.7 },
      { frequency: 2000, amplitude: 0.5 },
      { frequency: 5000, amplitude: 0.3 },
    ]
  }

  return (
    <div className="max-w-6xl mx-auto">
      <div className="text-center mb-8">
        <h1 className="text-3xl font-display font-bold text-racing-green mb-4">
          Analysis Results
        </h1>
        <p className="text-racing-green/70">
          Detailed frequency analysis for your audio file
        </p>
      </div>

      <div className="grid md:grid-cols-3 gap-6 mb-8">
        <div className="bg-white/70 backdrop-blur-sm rounded-xl p-6 shadow-lg">
          <h3 className="font-semibold text-racing-green mb-2">File Info</h3>
          <div className="space-y-2 text-sm">
            <p><span className="font-medium">Filename:</span> {mockData.filename}</p>
            <p><span className="font-medium">Duration:</span> {mockData.duration}</p>
            <p><span className="font-medium">Sample Rate:</span> {mockData.sampleRate} Hz</p>
          </div>
        </div>

        <div className="bg-white/70 backdrop-blur-sm rounded-xl p-6 shadow-lg md:col-span-2">
          <h3 className="font-semibold text-racing-green mb-4">Frequency Analysis</h3>
          <FrequencyChart data={mockData.frequencies} />
        </div>
      </div>

      <div className="text-center">
        <p className="text-racing-green/60 text-sm">
          Analysis ID: {id}
        </p>
      </div>
    </div>
  )
}

export default Results

