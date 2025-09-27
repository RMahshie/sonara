import { useParams, useNavigate } from 'react-router-dom'
import { useEffect } from 'react'
import ProgressBar from './ProgressBar'
import { useAnalysisStatus } from '../hooks/useAnalysisStatus'

const Analysis = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { status, error, isComplete, progress, message } = useAnalysisStatus(id || null)

  useEffect(() => {
    if (isComplete) {
      navigate(`/results/${id}`)
    }
  }, [isComplete, navigate, id])

  if (error) {
    return (
      <div className="max-w-2xl mx-auto">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-display font-bold text-racing-green mb-4">
            Analysis Error
          </h1>
          <p className="text-red-600">{error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="text-center mb-8">
        <h1 className="text-3xl font-display font-bold text-racing-green mb-4">
          Analysis in Progress
        </h1>
        <p className="text-racing-green/70">
          {message || 'Analyzing your audio file... This may take a few moments.'}
        </p>
      </div>

      <div className="bg-white/70 backdrop-blur-sm rounded-2xl p-8 shadow-xl">
        <div className="space-y-6">
          <div>
            <div className="flex justify-between text-sm text-racing-green/70 mb-2">
              <span>{status?.status === 'pending' ? 'Queued for processing...' :
                     status?.status === 'processing' ? 'Processing audio...' :
                     'Preparing results...'}</span>
              <span>{progress}%</span>
            </div>
            <ProgressBar progress={progress} />
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

