import React, { useState, useCallback, useEffect, useRef } from 'react';
import MindMap from './components/MindMap';
import SearchComponent from './components/SearchComponent';
import './App.css';

function App() {
  const [functionData, setFunctionData] = useState([]);
  const [selectedNode, setSelectedNode] = useState(null);
  const [isServerConnected, setIsServerConnected] = useState(false);
  const appRef = useRef(null);

  // Check server connection on component mount
  useEffect(() => {
    const checkServerConnection = async () => {
      try {
        const response = await fetch('http://localhost:8080/api/functions?page=1&pageSize=1');
        setIsServerConnected(response.ok);
      } catch (error) {
        console.error('Server connection failed:', error);
        setIsServerConnected(false);
      }
    };

    checkServerConnection();
  }, []);

  const handleFunctionSelect = useCallback((functionDataArray) => {
    setFunctionData(functionDataArray);
    setSelectedNode(null);
  }, []);

  const handleClearCanvas = useCallback(() => {
    setFunctionData([]);
    setSelectedNode(null);
  }, []);

  return (
    <div 
      className="App"
      ref={appRef}
    >
      <header className="app-header">
        <h1>Function Mind Map</h1>
        <div className="header-content">
          <p>Visualize your Go application's function call hierarchy</p>
          
          {!isServerConnected && (
            <div className="server-warning">
              ⚠️ Server not connected. Please ensure the Go server is running on localhost:8080
            </div>
          )}
          
          <SearchComponent 
            onFunctionSelect={handleFunctionSelect}
            onClearCanvas={handleClearCanvas}
          />
        </div>
      </header>
      
      <main className="app-main">
        {functionData.length === 0 ? (
          <div className="welcome-message">
            <h2>Welcome to Function Mind Map</h2>
            <p>Search for a function above to start exploring your codebase</p>
          </div>
        ) : (
          <MindMap 
            data={functionData} 
            selectedNode={selectedNode}
            onNodeSelect={setSelectedNode}
          />
        )}
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