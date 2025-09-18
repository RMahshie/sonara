import { useParams } from 'react-router-dom'
import ProgressBar from './ProgressBar'

const Analysis = () => {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="max-w-2xl mx-auto">
      <div className="text-center mb-8">
        <h1 className="text-3xl font-display font-bold text-racing-green mb-4">
          Analysis in Progress
        </h1>
        <p className="text-racing-green/70">
          Analyzing your audio file... This may take a few moments.
        </p>
      </div>

      <div className="bg-white/70 backdrop-blur-sm rounded-2xl p-8 shadow-xl">
        <div className="space-y-6">
          <div>
            <div className="flex justify-between text-sm text-racing-green/70 mb-2">
              <span>Processing audio...</span>
              <span>75%</span>
            </div>
            <ProgressBar progress={75} />
          </div>

          <div className="text-center">
            <p className="text-racing-green/60 text-sm">
              Analysis ID: {id}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Analysis
