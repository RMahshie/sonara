import { useParams } from 'react-router-dom'
import { useState, useEffect } from 'react'
import FrequencyChart from './FrequencyChart'
import { analysisService } from '../services/analysisService'

const Results = () => {
  const { id } = useParams<{ id: string }>()
  const [result, setResult] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // CSV conversion utility function
  const convertToCSV = (frequencyData: Array<{frequency: number, magnitude: number}>): string => {
    const headers = ['Frequency (Hz)', 'Magnitude (dB)']
    const rows = frequencyData.map(point => [point.frequency.toString(), point.magnitude.toString()])
    const csvContent = [headers, ...rows].map(row => row.join(',')).join('\n')
    return csvContent
  }

  // Download handler function
  const handleDownloadCSV = (frequencyData: Array<{frequency: number, magnitude: number}>, analysisId: string) => {
    if (!frequencyData || frequencyData.length === 0) {
      alert('No frequency data available for download')
      return
    }

    const csvContent = convertToCSV(frequencyData)
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')

    if (link.download !== undefined) {
      const url = URL.createObjectURL(blob)
      link.setAttribute('href', url)
      link.setAttribute('download', `analysis-${analysisId}-frequency-response.csv`)
      link.style.visibility = 'hidden'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url) // Clean up the URL object
    }
  }

  useEffect(() => {
    const fetchResults = async () => {
      try {
        const data = await analysisService.getAnalysisResults(id!)
        setResult(data)
      } catch (err: any) {
        setError(err.message || 'Failed to load results')
      } finally {
        setLoading(false)
      }
    }

    if (id) {
      fetchResults()
    }
  }, [id])

  if (loading) {
    return (
      <div className="max-w-6xl mx-auto">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-display font-bold text-racing-green mb-4">
            Loading Results...
          </h1>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="max-w-6xl mx-auto">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-display font-bold text-racing-green mb-4">
            Error Loading Results
          </h1>
          <p className="text-red-600">{error}</p>
        </div>
      </div>
    )
  }

  if (!result) {
    return (
      <div className="max-w-6xl mx-auto">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-display font-bold text-racing-green mb-4">
            No Results Found
          </h1>
        </div>
      </div>
    )
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

      <div className="grid md:grid-cols-4 gap-6 mb-8">
        <div className="bg-white/70 backdrop-blur-sm rounded-xl p-6 shadow-lg">
          <h3 className="font-semibold text-racing-green mb-2">Analysis Info</h3>
          <div className="space-y-2 text-sm">
            <p><span className="font-medium">Analysis ID:</span> {result.id}</p>
            {result.rt60 && <p><span className="font-medium">RT60:</span> {result.rt60.toFixed(2)}s</p>}
            {result.room_modes && result.room_modes.length > 0 && (
              <p><span className="font-medium">Room Modes:</span> {result.room_modes.map((mode: number) => mode.toFixed(0)).join(', ')} Hz</p>
            )}
            <p><span className="font-medium">Created:</span> {new Date(result.created_at).toLocaleString()}</p>
          </div>

          {result.frequency_data && result.frequency_data.length > 0 && (
            <div className="pt-2 border-t border-racing-green/10">
              <button
                onClick={() => handleDownloadCSV(result.frequency_data, result.id)}
                className="w-full px-3 py-2 text-sm bg-racing-green hover:bg-racing-green/90 text-cream font-medium rounded-md transition-colors duration-200"
              >
                Download Frequency Data (CSV)
              </button>
            </div>
          )}
        </div>

        <div className="bg-white/70 backdrop-blur-sm rounded-xl p-6 shadow-lg md:col-span-3">
          <h3 className="font-semibold text-racing-green mb-4">Frequency Response</h3>
          <FrequencyChart data={result.frequency_data || []} />
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

