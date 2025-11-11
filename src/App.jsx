import React from 'react';
import { BrowserRouter as Router, Routes, Route, NavLink, useLocation } from 'react-router-dom';
import MainPage from './components/MainPage';
import Presentation from './components/Presentation';
import './App.css';

const NavLinks = () => {
  return (
    <nav className="navbar">
      <NavLink to="/" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>Home</NavLink>
      <NavLink to="/presentation" className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}>Presentation</NavLink>
    </nav>
  );
};

function App() {
  return (
    <Router>
      <div className="app">
        <NavLinks />
        <Routes>
          <Route path="/" element={<MainPage />} />
          <Route path="/presentation" element={<Presentation />} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;

