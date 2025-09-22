import LiveRecorder from './LiveRecorder'

const Home = () => {
  return (
    <div className="max-w-4xl mx-auto">
      <div className="text-center mb-12">
        <h1 className="text-4xl md:text-6xl font-display font-bold text-racing-green mb-4">
          Sonara
        </h1>
        <p className="text-xl text-racing-green/80 max-w-2xl mx-auto">
          Advanced audio analysis for music professionals.
          Analyze your room acoustics with professional-grade measurements.
        </p>
      </div>

      <div className="bg-white/70 backdrop-blur-sm rounded-2xl p-8 shadow-xl">
        <LiveRecorder />
      </div>
    </div>
  )
}

export default Home

