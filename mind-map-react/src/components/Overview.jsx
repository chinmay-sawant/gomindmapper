import React from 'react';
import Navbar from './Navbar';
import './Overview.css';

const Overview = () => {
  return (
    <div className="overview-container">
      <Navbar />

      <header className="hero-section">
        <div className="badge">Alpha</div>
        <h1>Understand your Go code<br />through an interactive mind map.</h1>
        <p className="lead">
          GoMindMapper scans a Go repository, filters noise & framework chatter, then serves a navigable function call graph.
          Switch between offline JSON snapshots or a live API with pagination across entrypoint roots.
        </p>
        <div className="button-row">
          <a href="/view/" className="btn primary">Launch Mind Map →</a>
          <a href="https://github.com/chinmay-sawant/gomindmapper" target="_blank" rel="noreferrer" className="btn">GitHub</a>
        </div>
      </header>

      <main className="main-content">
        <section className="why-section">
          <h2>Why</h2>
          <div className="two-col">
            <div className="border-block">
              <p>
                Reading a large Go service by hopping files is slow. Architects want a high‑level dependency picture while contributors need local, expandable context.
                Existing tools either over‑simplify (just a list) or overwhelm (full static graph). GoMindMapper returns curated slices: a page of root functions plus the full closure below each—interactive & incremental.
              </p>
            </div>
            <div className="border-block">
              <ul className="feat-list">
                <li>Noise reduction: user→user edges only</li>
                <li>Pagination across true entrypoints</li>
                <li>Zoom / pan / expand drill‑down</li>
                <li>Offline JSON or live API mode</li>
                <li>Dark ergonomic interface</li>
                <li>Future: search, metrics, exports</li>
              </ul>
            </div>
          </div>
        </section>

        <section className="quick-start-section">
          <h2>Quick Start</h2>
          <div className="two-col">
            <div className="card">
              <h3>1. Build & Run</h3>
              <pre><code>go run cmd/server/main.go -path . -addr :8080

# (optional) React build
cd mind-map-react
npm install
npm run build

# browse http://localhost:8080/</code></pre>
            </div>
            <div className="card">
              <h3>2. Use the Map</h3>
              <p>
                Toggle <code className="inline-code">Use Live Server</code> for paginated data or upload a previously generated <code className="inline-code">functionmap.json</code>.
                Pan (drag background), zoom (wheel), expand nodes, inspect details on the side panel.
              </p>
            </div>
          </div>
        </section>

        <section className="architecture-section">
          <h2>Architecture</h2>
          <div className="card">
            <pre><code>Analyzer (cmd/main.go)
  ↳ writes functions.json / functionmap.json
Server (cmd/server/main.go)
  ↳ in-memory cache, pagination, reload
React (/view)
  ↳ mind map explorer (BrowserRouter basename)</code></pre>
          </div>
        </section>

        <section className="roadmap-section">
          <h2>Roadmap (Excerpt)</h2>
          <div className="grid">
            <div className="card">
              <h3>Search</h3>
              <p>Endpoint + UI for name / fuzzy lookup of functions; focus map on match.</p>
            </div>
            <div className="card">
              <h3>Metrics Overlay</h3>
              <p>Display fan‑in / fan‑out badges & heat colors for hotspots.</p>
            </div>
            <div className="card">
              <h3>Incremental Watch</h3>
              <p>File watcher updates only changed functions to keep cache fresh.</p>
            </div>
            <div className="card">
              <h3>Export Formats</h3>
              <p>GraphML / DOT / CSV for external analysis or knowledge bases.</p>
            </div>
          </div>
        </section>
      </main>

      <footer className="footer">
        MIT Licensed • <a className="inline-link" href="https://github.com/chinmay-sawant/gomindmapper" target="_blank" rel="noreferrer">GitHub Repo</a>
      </footer>
    </div>
  );
};

export default Overview;