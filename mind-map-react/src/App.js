import React, { useState, useCallback, useEffect, useRef } from 'react';
import MindMap from './components/MindMap';
import './App.css';

// Default EmployeeApp data to show by default
const defaultEmployeeAppData = [
  {
    "name": "main.main",
    "line": 9,
    "filePath": "EmployeeApp\\main.go",
    "called": [
      {
        "name": "config.Load",
        "line": 9,
        "filePath": "EmployeeApp\\internal\\config\\config.go"
      },
      {
        "name": "routes.SetupRouter",
        "line": 10,
        "filePath": "EmployeeApp\\internal\\routes\\routes.go"
      }
    ]
  },
  {
    "name": "routes.SetupRouter",
    "line": 10,
    "filePath": "EmployeeApp\\internal\\routes\\routes.go",
    "called": [
      {
        "name": "middleware.CORS",
        "line": 5,
        "filePath": "EmployeeApp\\internal\\middleware\\cors.go"
      },
      {
        "name": "middleware.Logger",
        "line": 10,
        "filePath": "EmployeeApp\\internal\\middleware\\logger.go"
      },
      {
        "name": "handlers.NewEmployeeHandler",
        "line": 12,
        "filePath": "EmployeeApp\\internal\\handlers\\employee.go"
      }
    ]
  }
];

function App() {
  const [functionData, setFunctionData] = useState(defaultEmployeeAppData);
  const [selectedNode, setSelectedNode] = useState(null);
  const [fileName, setFileName] = useState('EmployeeApp (Default)');
  const [dragActive, setDragActive] = useState(false);
  const [dragCounter, setDragCounter] = useState(0);
  const appRef = useRef(null);

  const handleFileUpload = useCallback((file) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const jsonData = JSON.parse(e.target.result);
        setFunctionData(jsonData);
        setFileName(file.name);
        setSelectedNode(null); // Clear selection when new data loads
      } catch (error) {
        alert('Error parsing JSON file: ' + error.message);
      }
    };
    reader.readAsText(file);
  }, []);

  const handleDrop = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    setDragCounter(0);
    
    const files = [...e.dataTransfer.files];
    if (files && files[0]) {
      const file = files[0];
      if (file.type === 'application/json' || file.name.endsWith('.json')) {
        handleFileUpload(file);
      } else {
        alert('Please upload a JSON file');
      }
    }
  }, [handleFileUpload]);

  const handleDragEnter = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragCounter(prev => prev + 1);
    if (!dragActive) {
      setDragActive(true);
    }
  }, [dragActive]);

  const handleDragLeave = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragCounter(prev => {
      const newCounter = prev - 1;
      // Only hide overlay when counter reaches 0 (actually left the container)
      if (newCounter === 0) {
        // Add a small delay to prevent flickering
        setTimeout(() => {
          setDragActive(false);
        }, 100);
      }
      return newCounter;
    });
  }, []);

  const handleDragOver = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleFileSelect = useCallback((e) => {
    const file = e.target.files[0];
    if (file) {
      handleFileUpload(file);
    }
  }, [handleFileUpload]);

  // Set up drag event listeners with non-passive option to allow preventDefault
  useEffect(() => {
    const appElement = appRef.current;
    if (!appElement) return;

    const handleDropNonPassive = (e) => handleDrop(e);
    const handleDragEnterNonPassive = (e) => handleDragEnter(e);
    const handleDragLeaveNonPassive = (e) => handleDragLeave(e);
    const handleDragOverNonPassive = (e) => handleDragOver(e);

    // Add event listeners with passive: false to allow preventDefault
    appElement.addEventListener('drop', handleDropNonPassive, { passive: false });
    appElement.addEventListener('dragenter', handleDragEnterNonPassive, { passive: false });
    appElement.addEventListener('dragleave', handleDragLeaveNonPassive, { passive: false });
    appElement.addEventListener('dragover', handleDragOverNonPassive, { passive: false });

    // Cleanup function
    return () => {
      appElement.removeEventListener('drop', handleDropNonPassive);
      appElement.removeEventListener('dragenter', handleDragEnterNonPassive);
      appElement.removeEventListener('dragleave', handleDragLeaveNonPassive);
      appElement.removeEventListener('dragover', handleDragOverNonPassive);
    };
  }, [handleDrop, handleDragEnter, handleDragLeave, handleDragOver]);

  return (
    <div 
      className="App"
      ref={appRef}
    >
      <header className="app-header">
        <h1>Function Mind Map</h1>
        <div className="header-content">
          <p>Visualize your Go application's function call hierarchy</p>
          <div className="file-input-section">
            <span className="current-file">Current: {fileName}</span>
            <label htmlFor="file-input" className="file-input-label">
              Choose JSON File
            </label>
            <input
              id="file-input"
              type="file"
              accept=".json,application/json"
              onChange={handleFileSelect}
              className="file-input"
            />
          </div>
        </div>
      </header>
      
      {dragActive && (
        <div className="drag-overlay">
          <div className="drag-message">
            <h2>Drop JSON file here</h2>
            <p>Release to load function map data</p>
          </div>
        </div>
      )}
      
      <main className="app-main">
        <MindMap 
          data={functionData} 
          selectedNode={selectedNode}
          onNodeSelect={setSelectedNode}
        />
      </main>
      
      {selectedNode && (
        <div className="node-details">
          <h3>Function Details</h3>
          <p><strong>Name:</strong> {selectedNode.name}</p>
          <p><strong>Line:</strong> {selectedNode.line}</p>
          <p><strong>File:</strong> {selectedNode.filePath}</p>
          {selectedNode.called && selectedNode.called.length > 0 && (
            <div>
              <strong>Calls:</strong>
              <ul>
                {selectedNode.called.map((fn, index) => (
                  <li key={index}>{fn.name} (line {fn.line})</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default App;