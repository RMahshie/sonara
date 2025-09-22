import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { vi } from 'vitest'
import LiveRecorder from '../LiveRecorder'

// Mock axios first
vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => ({
      post: vi.fn(() => Promise.resolve({
        data: { id: 'test-analysis-id', upload_url: 'test-upload-url' }
      }))
    })),
    put: vi.fn(() => Promise.resolve())
  }
}))

// Mock the API service
vi.mock('../services/api', () => ({
  api: {
    post: vi.fn(() => Promise.resolve({
      data: { id: 'test-analysis-id', upload_url: 'test-upload-url' }
    }))
  }
}))

// Mock navigator.mediaDevices
const mockGetUserMedia = vi.fn(() => Promise.resolve({
  getTracks: vi.fn(() => [
    { stop: vi.fn() },
    { stop: vi.fn() }
  ])
}))

Object.defineProperty(navigator, 'mediaDevices', {
  value: {
    getUserMedia: mockGetUserMedia
  },
  writable: true
})

// Mock MediaRecorder
global.MediaRecorder = vi.fn().mockImplementation(() => ({
  start: vi.fn(),
  stop: vi.fn(),
  ondataavailable: null,
  onstop: null
})) as any

// Add static method
global.MediaRecorder.isTypeSupported = vi.fn(() => true)

// Mock Audio
global.Audio = vi.fn().mockImplementation(() => ({
  play: vi.fn(() => Promise.resolve()),
  pause: vi.fn(),
  currentTime: 0,
  duration: 10,
  ontimeupdate: null,
  onended: null
}))

describe('LiveRecorder', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders recording interface', () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    expect(screen.getByText('Room Acoustic Analysis')).toBeInTheDocument()
    expect(screen.getByText('Click to analyze your room acoustics')).toBeInTheDocument()
    expect(screen.getByText('Analyze Room')).toBeInTheDocument()
  })

  it('has correct styling classes', () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    // Find the main container div with border classes
    const container = document.querySelector('.border-dashed')
    expect(container).toHaveClass('border-2', 'border-dashed', 'rounded-xl')
  })

  it('renders microphone icon', () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    expect(screen.getByText('🎤')).toBeInTheDocument()
  })

  it('shows setup instructions', () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    expect(screen.getByText('Quick Setup:')).toBeInTheDocument()
    expect(screen.getByText(/Set speakers to normal listening volume/)).toBeInTheDocument()
    expect(screen.getByText(/Position microphone at listening position/)).toBeInTheDocument()
  })

  it('handles microphone access', async () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    const analyzeButton = screen.getByText('Analyze Room')
    fireEvent.click(analyzeButton)

    await waitFor(() => {
      expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledWith({
        audio: {
          echoCancellation: false,
          noiseSuppression: false,
          autoGainControl: false,
          sampleRate: 48000
        }
      })
    })
  })

  it('handles microphone permission denied', async () => {
    // Mock permission denied
    mockGetUserMedia.mockRejectedValueOnce(
      new Error('Permission denied')
    )

    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    const analyzeButton = screen.getByText('Analyze Room')
    fireEvent.click(analyzeButton)

    await waitFor(() => {
      expect(screen.getByText('Permission denied')).toBeInTheDocument()
    })
  })

  it('shows recording progress', async () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    const analyzeButton = screen.getByText('Analyze Room')
    fireEvent.click(analyzeButton)

    await waitFor(() => {
      expect(screen.getByText('Recording in progress...')).toBeInTheDocument()
    })
  })

  it('has cancel recording button during recording', async () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    const analyzeButton = screen.getByText('Analyze Room')
    fireEvent.click(analyzeButton)

    await waitFor(() => {
      expect(screen.getByText('Cancel Recording')).toBeInTheDocument()
    })
  })

  it('uses btn-primary class for main button', () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    const analyzeButton = screen.getByText('Analyze Room')
    expect(analyzeButton).toHaveClass('btn-primary')
  })

  it('hides analyze button during recording', async () => {
    render(
      <BrowserRouter>
        <LiveRecorder />
      </BrowserRouter>
    )

    const analyzeButton = screen.getByText('Analyze Room')
    fireEvent.click(analyzeButton)

    await waitFor(() => {
      // During recording, the Analyze Room button should not be present
      expect(screen.queryByText('Analyze Room')).not.toBeInTheDocument()
      // Instead we should see the recording UI
      expect(screen.getByText('Recording in progress...')).toBeInTheDocument()
    })
  })
})