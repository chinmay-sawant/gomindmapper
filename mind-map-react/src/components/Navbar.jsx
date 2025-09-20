import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import './Navbar.css';

const repoUrl = 'https://github.com/chinmay-sawant/gomindmapper';
const isProduction = process.env.NODE_ENV === 'production';

export default function Navbar({ onReload }) {
  const loc = useLocation();
  const isActive = (to) => loc.pathname === to || (to !== '/' && loc.pathname.startsWith(to));
  return (
    <nav className="nav-root">
      <div className="nav-left">
        <Link to="/" className="nav-brand">GoMindMapper</Link>
        <div className="nav-links">
          {isProduction && <a href="/" className={isActive('/') ? 'active' : ''}>Overview</a>}
          <Link to="/" className={isActive('/') ? 'active' : ''}>Mind Map</Link>
          <a href={repoUrl} target="_blank" rel="noreferrer" className="external-link">GitHub</a>
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
