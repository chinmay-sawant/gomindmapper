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
          <a href="/gomindmapper/view" className="btn primary">Launch Mind Map →</a>
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

          <div className="quick-start-grid">
            <div className="primary-card card">
              <h3>Run the application (single command)</h3>
              <p className="lead muted">Run the server against a target repository/subfolder — this example uses <code>gopdfsuit</code>:</p>
              <pre className="cmd"><code>go run cmd/server/main.go -path gopdfsuit -addr :8080 --include-external=true --skip-folders="golang.org,gin-gonic,bytedance,ugorji,go-playground"</code></pre>
              <div className="flag-list">
                <strong>Flags</strong>
                <ul>
                  <li><code>-path &lt;dir&gt;</code>: repository/subfolder to analyze (e.g., <code>gopdfsuit</code>).</li>
                  <li><code>-addr &lt;addr&gt;</code>: HTTP server address (default <code>:8080</code>).</li>
                  <li><code>--include-external</code>: include external module functions in the graph.</li>
                  <li><code>--skip-folders</code>: comma-separated dependency prefixes to ignore during external scanning.</li>
                </ul>
              </div>
              <p className="muted">Note: production React assets are built into <code>/docs</code> (see <code>mind-map-react/vite.config.js</code>) and are served by the Go server — you don't need to run the React project separately for production.</p>
            </div>

            <div className="aux-cards">
              <div className="card">
                <h4>Build & Run (production)</h4>
                <pre><code>cd &lt;repo-root&gt;
go run cmd/server/main.go -path . -addr :8080</code></pre>
                <p className="muted">Starts the Go server which serves the Overview at <code>/gomindmapper/</code> and the app at <code>/gomindmapper/view/</code>.</p>
              </div>

              <div className="card">
                <h4>Build Frontend (optional)</h4>
                <pre><code>cd mind-map-react
npm install
npm run build</code></pre>
                <p className="muted">Places production files into <code>../docs</code> for the Go server to serve.</p>
              </div>

              <div className="card">
                <h4>Development Mode</h4>
                <pre><code>// Terminal 1
cd mind-map-react
npm install
npm run dev

// Terminal 2
cd &lt;repo-root&gt;
go run cmd/server/main.go -path . -addr :8080</code></pre>
                <p className="muted">Use Vite dev server for UI hot-reload while the Go server provides live data. Open <code>http://localhost:5173/gomindmapper/view</code>.</p>
              </div>

              <div className="card">
                <h4>CLI-only Analysis</h4>
                <pre><code>go run cmd/main.go -path . --include-external=true</code></pre>
                <p className="muted">Generates <code>functions.json</code>, <code>functionmap.json</code> and <code>removed_calls.json</code> for offline analysis.</p>
              </div>

              <div className="card">
                <h4>Makefile Shortcuts</h4>
                <pre><code>make ui-build   # builds the React app
make server     # runs the Go server</code></pre>
                <p className="muted">Convenient targets to build and run the project.</p>
              </div>
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