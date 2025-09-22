import { Link } from 'react-router-dom'

const Header = () => {
  return (
    <header className="bg-racing-green text-cream shadow-lg">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <Link to="/" className="text-2xl font-display font-bold hover:text-brass transition-colors">
            Sonara
          </Link>
          <nav className="hidden md:flex space-x-8">
            <Link
              to="/"
              className="hover:text-brass transition-colors font-medium"
            >
              Home
            </Link>
          </nav>
        </div>
      </div>
    </header>
  )
}

export default Header

