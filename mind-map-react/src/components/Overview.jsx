import React from 'react';
import Navbar from './Navbar';
import ScreenshotSlideshow from './ScreenshotSlideshow';
import ComparisonTable from './ComparisonTable';
import './Overview.css';

const Overview = () => {
  return (
    <div className="overview-container">
      <Navbar />

      <div className="overview-content">
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

      <ScreenshotSlideshow />

      <ComparisonTable />

      <main className="main-content">
        <section className="why-section">
          <h2>Why</h2>
          <div className="two-col">
            <div className="border-block">
              <p>
                Earlier I had created <a href="https://github.com/chinmay-sawant/CodeMapper" target="_blank" rel="noreferrer" className="inline-link">CodeMapper</a> as a code visualizer using the react-flow library with a Go backend using AST parsing. 
                When I tried running it on my organization's custom codebase, it didn't provide the desired output. 
                So I decided to create GoMindMapper from scratch with improved functionality - custom-built nodes inspired by Google NotebookLLM, better user control with pagination, and the ability to search any directory location.
              </p>
              <p>
                Hopefully you'll like this tool and enjoy using it! If you do, please provide a star on <a href="https://github.com/chinmay-sawant/gomindmapper" target="_blank" rel="noreferrer" className="inline-link">GitHub</a>.
              </p>
            </div>
            <div className="border-block">
              <h3>Key Improvements Over CodeMapper:</h3>
              <ul className="feat-list">
                <li><strong>Custom-built nodes</strong> inspired by Google NotebookLLM for better visualization</li>
                <li><strong>Pagination support</strong> for handling large codebases efficiently</li>
                <li><strong>Directory location flexibility</strong> - search any directory you want</li>
                <li><strong>Built from scratch</strong> with improved AST parsing and analysis</li>
                <li><strong>Better user control</strong> with interactive mind maps and drill-down</li>
                <li><strong>Noise reduction</strong> focusing on user-to-user function relationships</li>
              </ul>
            </div>
          </div>
        </section>

        <section className="quick-start-section">
          <h2>Quick Start</h2>
          <div className="two-col">
            <div className="card">
              <h3>1. Build & Run</h3>
              <pre><code>go run cmd/server/main.go -path . -addr :8080</code></pre>
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
      </main>

      <footer className="footer">
        Made with ❤️ in India • <a className="inline-link" href="https://github.com/chinmay-sawant" target="_blank" rel="noreferrer">Visit GitHub</a>
      </footer>
      </div>
    </div>
  );
};

export default Overview;