import React from 'react';
import './ComparisonTable.css';

const ComparisonTable = () => {
  const tools = [
    {
      name: 'GoMindMapper',
      type: 'Function Relationship Mind Map',
      output: 'Interactive web-based mind map (React)',
      interactivity: 'High - pannable, zoomable, expandable nodes, pagination',
      filtering: 'Filters noise, stdlib, frameworks; focuses on user-to-user calls',
      analysis: 'Partial AST parsing, call graph extraction',
      keyFeatures: 'Live server with pagination, offline JSON upload, custom nodes, directory flexibility, drag drop existing json data'
    },
    {
      name: 'go-callvis',
      type: 'Call Graph Visualization',
      output: 'Graphviz dot format, SVG/PNG static images, interactive web viewer',
      interactivity: 'Medium - interactive viewer for focusing packages',
      filtering: 'Focus packages, group by package/type, ignore stdlib, filter prefixes',
      analysis: 'Pointer analysis for call graph',
      keyFeatures: 'Web server mode, static exports, various grouping options'
    },
    {
      name: 'godepgraph',
      type: 'Package Dependency Graph',
      output: 'Graphviz dot or Mermaid format',
      interactivity: 'Low - static graph output',
      filtering: 'Ignore stdlib, vendored packages, by name/prefix',
      analysis: 'Package import relationships',
      keyFeatures: 'Color-coded packages (stdlib, vendored, cgo), supports go mod'
    },
    {
      name: 'goda',
      type: 'Dependency Analysis Toolkit',
      output: 'Text lists, graphs, trees, stats',
      interactivity: 'Low - command-line queries',
      filtering: 'Complex expressions, set operations, tags',
      analysis: 'Advanced dependency queries, symbol weighting',
      keyFeatures: 'Arithmetic on package sets, build stats, cut analysis, weight analysis'
    }
  ];

  return (
    <section className="comparison-section">
      <h2>Comparison with Other Go Code Visualizers</h2>
      <div className="table-container">
        <table className="comparison-table">
          <thead>
            <tr>
              <th>Tool</th>
              <th>Type</th>
              <th>Output</th>
              <th>Interactivity</th>
              <th>Filtering</th>
              <th>Analysis</th>
              <th>Key Features</th>
            </tr>
          </thead>
          <tbody>
            {tools.map((tool, index) => (
              <tr key={index} className={tool.name === 'GoMindMapper' ? 'highlight' : ''}>
                <td className="tool-name">{tool.name}</td>
                <td>{tool.type}</td>
                <td>{tool.output}</td>
                <td>{tool.interactivity}</td>
                <td>{tool.filtering}</td>
                <td>{tool.analysis}</td>
                <td>{tool.keyFeatures}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
};

export default ComparisonTable;