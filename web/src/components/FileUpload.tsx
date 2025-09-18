import { useCallback, useState } from 'react'
import { useDropzone } from 'react-dropzone'
import { useNavigate } from 'react-router-dom'
import ProgressBar from './ProgressBar'

const FileUpload = () => {
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const navigate = useNavigate()

  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    const file = acceptedFiles[0]
    if (!file) return

    setUploading(true)
    setProgress(0)

    // Simulate upload progress
    const interval = setInterval(() => {
      setProgress(prev => {
        if (prev >= 90) {
          clearInterval(interval)
          // Simulate API call completion
          setTimeout(() => {
            setUploading(false)
            // Navigate to analysis page with mock ID
            navigate('/analysis/123')
          }, 500)
          return 100
        }
        return prev + 10
      })
    }, 200)

    // TODO: Implement actual file upload to API
    console.log('Uploading file:', file.name)
  }, [navigate])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'audio/*': ['.wav', '.mp3', '.flac', '.aac', '.ogg']
    },
    multiple: false,
    disabled: uploading
  })

  return (
    <div className="w-full">
      <div
        {...getRootProps()}
        className={`
          border-2 border-dashed rounded-xl p-12 text-center cursor-pointer transition-all duration-200
          ${isDragActive
            ? 'border-brass bg-brass/10'
            : 'border-racing-green/30 hover:border-racing-green/60'
          }
          ${uploading ? 'pointer-events-none opacity-50' : ''}
        `}
      >
        <input {...getInputProps()} />

        {uploading ? (
          <div className="space-y-4">
            <div className="text-racing-green font-medium">Uploading...</div>
            <ProgressBar progress={progress} />
          </div>
        ) : (
          <div className="space-y-4">
            <div className="text-6xl text-racing-green/40">♪</div>
            <div>
              <p className="text-xl font-medium text-racing-green mb-2">
                {isDragActive ? 'Drop your audio file here' : 'Drag & drop your audio file'}
              </p>
              <p className="text-racing-green/60">
                or click to browse (WAV, MP3, FLAC, AAC, OGG)
              </p>
            </div>
            <button
              type="button"
              className="btn-primary mt-4"
              onClick={(e) => e.stopPropagation()}
            >
              Choose File
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default FileUpload