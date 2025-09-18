import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import FileUpload from '../FileUpload'

describe('FileUpload', () => {
  it('renders upload area', () => {
    render(
      <BrowserRouter>
        <FileUpload />
      </BrowserRouter>
    )

    expect(screen.getByText('Drag & drop your audio file')).toBeInTheDocument()
    expect(screen.getByText('Choose File')).toBeInTheDocument()
    expect(screen.getByText('or click to browse (WAV, MP3, FLAC, AAC, OGG)')).toBeInTheDocument()
  })

  it('has correct styling classes', () => {
    render(
      <BrowserRouter>
        <FileUpload />
      </BrowserRouter>
    )

    // Find the dropzone by its role
    const dropzone = screen.getByRole('presentation')
    expect(dropzone).toHaveClass('border-2', 'border-dashed', 'rounded-xl', 'cursor-pointer')
  })

  it('renders music note icon', () => {
    render(
      <BrowserRouter>
        <FileUpload />
      </BrowserRouter>
    )

    expect(screen.getByText('♪')).toBeInTheDocument()
  })
})