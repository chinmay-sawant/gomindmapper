import React, { useState, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useTheme } from '../contexts/ThemeContext.jsx';
import './Navbar.css';

const repoUrl = 'https://github.com/chinmay-sawant/gomindmapper';

const ThemeToggle = () => {
  const { theme, toggleTheme } = useTheme();
  
  return (
    <button 
      className="theme-toggle" 
      onClick={toggleTheme}
      title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
    >
      {theme === 'light' ? (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 2.25a.75.75 0 01.75.75v2.25a.75.75 0 01-1.5 0V3a.75.75 0 01.75-.75zM7.5 12a4.5 4.5 0 119 0 4.5 4.5 0 01-9 0zM18.894 6.166a.75.75 0 00-1.06-1.06l-1.591 1.59a.75.75 0 101.06 1.061l1.591-1.59zM21.75 12a.75.75 0 01-.75.75h-2.25a.75.75 0 010-1.5H21a.75.75 0 01.75.75zM17.834 18.894a.75.75 0 001.06-1.06l-1.59-1.591a.75.75 0 10-1.061 1.06l1.59 1.591zM12 18a.75.75 0 01.75.75V21a.75.75 0 01-1.5 0v-2.25A.75.75 0 0112 18zM7.758 17.303a.75.75 0 00-1.061-1.06L5.106 17.834a.75.75 0 101.06 1.06l1.591-1.59zM6 12a.75.75 0 01-.75.75H3a.75.75 0 010-1.5h2.25A.75.75 0 016 12zM6.697 7.757a.75.75 0 001.06-1.06l-1.59-1.591a.75.75 0 00-1.061 1.06l1.59 1.591z" />
        </svg>
      ) : (
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
          <path d="M9.528 1.718a.75.75 0 01.162.819A8.97 8.97 0 009 6a9 9 0 009 9 8.97 8.97 0 003.463-.69.75.75 0 01.981.98 10.503 10.503 0 01-9.694 6.46c-5.799 0-10.5-4.701-10.5-10.5 0-4.368 2.667-8.112 6.46-9.694a.75.75 0 01.818.162z" />
        </svg>
      )}
    </button>
  );
};

export default function Navbar({ onReload, onDownload }) {
  const [stars, setStars] = useState(0);

  useEffect(() => {
    fetch('https://api.github.com/repos/chinmay-sawant/gomindmapper')
      .then(res => res.json())
      .then(data => setStars(data.stargazers_count || 0))
      .catch(() => setStars(0));
  }, []);

  const loc = useLocation();
  const isActive = (to) => loc.pathname === to || (to !== '/' && loc.pathname.startsWith(to));

  // Check if we're on the overview page
  const isOverview = loc.pathname === '/';

  if (isOverview) {
    return (
      <nav className="nav-root">
        <div className="nav-left">
          <Link to="/" className="nav-brand">GoMindMapper</Link>
          <div className="nav-links">
            <span className={isActive('/') ? 'active' : ''}>Overview</span>
            <Link to="/view" className={isActive('/view') ? 'active' : ''}>Mind Map</Link>
            <a href="https://github.com/chinmay-sawant/gomindmapper" target="_blank" rel="noreferrer" className="external-link">GitHub ⭐ {stars}</a>
          </div>
        </div>
        <div className="nav-right">
          <ThemeToggle />
          <Link to="/view" className="btn-small">Open Map →</Link>
        </div>
      </nav>
    );
  }

  return (
    <nav className="nav-root">
      <div className="nav-left">
        <Link to="/" className="nav-brand">GoMindMapper</Link>
        <div className="nav-links">
          <Link to="/" className={isActive('/') ? 'active' : ''}>Overview</Link>
          <span className={isActive('/view') ? 'active' : ''}>Mind Map</span>
          <a href="https://github.com/chinmay-sawant/gomindmapper" target="_blank" rel="noreferrer" className="external-link">GitHub ⭐ {stars}</a>
        </div>
      </div>
      <div className="nav-right">
        <ThemeToggle />
        {onDownload && (
          <a href={onDownload} download="function_relations.json" className="download-btn" title="Download cached relations data">
            Download Data
          </a>
        )}
        {onReload && (
          <button className="reload-btn" onClick={onReload} title="POST /api/reload and refetch">
            Reload Scan
          </button>
        )}
      </div>
    </nav>
  );
}
