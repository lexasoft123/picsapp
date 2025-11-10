import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import MainPage from './components/MainPage';
import Presentation from './components/Presentation';
import './App.css';

function App() {
  return (
    <Router>
      <div className="app">
        <nav className="navbar">
          <Link to="/" className="nav-link">Home</Link>
          <Link to="/presentation" className="nav-link">Presentation</Link>
        </nav>
        <Routes>
          <Route path="/" element={<MainPage />} />
          <Route path="/presentation" element={<Presentation />} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;

