import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import './Navbar.css';

const repoUrl = 'https://github.com/chinmay-sawant/gomindmapper';

export default function Navbar({ onReload }) {
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
            <a href="https://github.com/chinmay-sawant/gomindmapper" target="_blank" rel="noreferrer" className="external-link">GitHub</a>
          </div>
        </div>
        <div className="nav-right">
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
          <a href="https://github.com/chinmay-sawant/gomindmapper" target="_blank" rel="noreferrer" className="external-link">GitHub</a>
        </div>
      </div>
      <div className="nav-right">
        {onReload && (
          <button className="reload-btn" onClick={onReload} title="POST /api/reload and refetch">
            Reload Scan
          </button>
        )}
      </div>
    </nav>
  );
}
