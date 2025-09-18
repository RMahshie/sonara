import { Routes, Route } from 'react-router-dom'
import Header from './components/Header'
import Home from './components/Home'
import Analysis from './components/Analysis'
import Results from './components/Results'

function App() {
  return (
    <div className="min-h-screen bg-cream">
      <Header />
      <main className="container mx-auto px-4 py-8">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/analysis/:id" element={<Analysis />} />
          <Route path="/results/:id" element={<Results />} />
        </Routes>
      </main>
    </div>
  )
}

export default App