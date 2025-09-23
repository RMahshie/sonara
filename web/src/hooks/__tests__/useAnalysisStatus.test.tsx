import { render, waitFor } from '@testing-library/react'
import { vi } from 'vitest'
import { useAnalysisStatus } from '../useAnalysisStatus'

// Mock axios
vi.mock('axios', () => ({
  default: {
    get: vi.fn()
  }
}))

// Import the mocked axios
import axios from 'axios'

// Test component that uses the hook
function TestComponent({ analysisId }: { analysisId: string | null }) {
  const { status, error, isComplete, isFailed, progress, message } = useAnalysisStatus(analysisId)

  return (
    <div>
      <div data-testid="status">{status?.status || 'null'}</div>
      <div data-testid="progress">{progress}</div>
      <div data-testid="message">{message || 'null'}</div>
      <div data-testid="isComplete">{isComplete ? 'true' : 'false'}</div>
      <div data-testid="isFailed">{isFailed ? 'true' : 'false'}</div>
      <div data-testid="error">{error || 'null'}</div>
    </div>
  )
}

describe('useAnalysisStatus', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not poll when analysisId is null', () => {
    render(<TestComponent analysisId={null} />)
    expect(axios.get).not.toHaveBeenCalled()
  })

  it('fetches status when analysisId is provided', async () => {
    const mockResponse = {
      data: {
        id: 'test-analysis',
        status: 'processing',
        progress: 25,
        message: 'Processing analysis...'
      }
    }

    vi.mocked(axios.get).mockResolvedValue(mockResponse)

    render(<TestComponent analysisId="test-analysis-id" />)

    await waitFor(() => {
      expect(axios.get).toHaveBeenCalledWith('/api/analyses/test-analysis-id/status')
      expect(document.querySelector('[data-testid="status"]')).toHaveTextContent('processing')
      expect(document.querySelector('[data-testid="progress"]')).toHaveTextContent('25')
      expect(document.querySelector('[data-testid="message"]')).toHaveTextContent('Processing analysis...')
    })
  })

  it('handles completed status correctly', async () => {
    const completedResponse = {
      data: {
        id: 'test-analysis',
        status: 'completed',
        progress: 100,
        message: 'Analysis complete!'
      }
    }

    vi.mocked(axios.get).mockResolvedValue(completedResponse)

    render(<TestComponent analysisId="test-analysis-id" />)

    await waitFor(() => {
      expect(document.querySelector('[data-testid="status"]')).toHaveTextContent('completed')
      expect(document.querySelector('[data-testid="isComplete"]')).toHaveTextContent('true')
      expect(document.querySelector('[data-testid="isFailed"]')).toHaveTextContent('false')
    })
  })

  it('handles failed status correctly', async () => {
    const failedResponse = {
      data: {
        id: 'test-analysis',
        status: 'failed',
        progress: 0,
        message: 'Analysis failed. Please try again.'
      }
    }

    vi.mocked(axios.get).mockResolvedValue(failedResponse)

    render(<TestComponent analysisId="test-analysis-id" />)

    await waitFor(() => {
      expect(document.querySelector('[data-testid="status"]')).toHaveTextContent('failed')
      expect(document.querySelector('[data-testid="isFailed"]')).toHaveTextContent('true')
      expect(document.querySelector('[data-testid="isComplete"]')).toHaveTextContent('false')
    })
  })

  it('handles API errors gracefully', async () => {
    vi.mocked(axios.get).mockRejectedValue(new Error('Network error'))

    render(<TestComponent analysisId="test-analysis-id" />)

    await waitFor(() => {
      expect(axios.get).toHaveBeenCalledTimes(1)
      expect(document.querySelector('[data-testid="error"]')).toHaveTextContent('Failed to fetch analysis status')
    })
  })

  it('provides correct default values', () => {
    render(<TestComponent analysisId={null} />)

    expect(document.querySelector('[data-testid="progress"]')).toHaveTextContent('0')
    expect(document.querySelector('[data-testid="isComplete"]')).toHaveTextContent('false')
    expect(document.querySelector('[data-testid="isFailed"]')).toHaveTextContent('false')
    expect(document.querySelector('[data-testid="error"]')).toHaveTextContent('null')
    expect(document.querySelector('[data-testid="status"]')).toHaveTextContent('null')
    expect(document.querySelector('[data-testid="message"]')).toHaveTextContent('null')
  })

  it('handles backend validation error for file too short', async () => {
    vi.mocked(axios.get).mockRejectedValue({
      response: { data: { message: 'Recording too short. Please ensure microphone is working.' } }
    })

    render(<TestComponent analysisId="test-id" />)

    await waitFor(() => {
      expect(document.querySelector('[data-testid="error"]')).toHaveTextContent('Recording too short. Please ensure microphone is working.')
    })
  })

  it('handles backend validation error for file too large', async () => {
    vi.mocked(axios.get).mockRejectedValue({
      response: { data: { message: 'Recording too large. Please try a shorter recording.' } }
    })

    render(<TestComponent analysisId="test-id" />)

    await waitFor(() => {
      expect(document.querySelector('[data-testid="error"]')).toHaveTextContent('Recording too large. Please try a shorter recording.')
    })
  })

  it('handles backend validation error for unsupported format', async () => {
    vi.mocked(axios.get).mockRejectedValue({
      response: { data: { message: 'Recording format not supported. Please try again.' } }
    })

    render(<TestComponent analysisId="test-id" />)

    await waitFor(() => {
      expect(document.querySelector('[data-testid="error"]')).toHaveTextContent('Recording format not supported. Please try again.')
    })
  })
})
