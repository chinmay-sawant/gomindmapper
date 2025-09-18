import React, { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import Node from './Node';
import './MindMap.css';

const MindMap = ({ data, selectedNode, onNodeSelect }) => {
  const [expandedNodes, setExpandedNodes] = useState(new Set());
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [lastMousePosition, setLastMousePosition] = useState({ x: 0, y: 0 });
  const containerRef = useRef(null);

  // Add global mouse event listeners for better dragging
  useEffect(() => {
    const handleGlobalMouseMove = (e) => {
      if (isDragging) {
        const deltaX = e.clientX - lastMousePosition.x;
        const deltaY = e.clientY - lastMousePosition.y;
        
        setPan(prev => ({
          x: prev.x + deltaX,
          y: prev.y + deltaY
        }));
        
        setLastMousePosition({ x: e.clientX, y: e.clientY });
      }
    };

    const handleGlobalMouseUp = () => {
      setIsDragging(false);
    };

    if (isDragging) {
      document.addEventListener('mousemove', handleGlobalMouseMove);
      document.addEventListener('mouseup', handleGlobalMouseUp);
    }

    return () => {
      document.removeEventListener('mousemove', handleGlobalMouseMove);
      document.removeEventListener('mouseup', handleGlobalMouseUp);
    };
  }, [isDragging, lastMousePosition]);

  // Build the tree structure from flat function data
  const treeData = useMemo(() => {
    const buildTree = (functions) => {
      const functionMap = new Map();
      const rootNodes = [];
      
      // Create unique key for each function based on name and file path
      const getUniqueKey = (fn) => `${fn.name}@${fn.filePath}`;
      
      // Create a map of all functions with unique keys
      functions.forEach(fn => {
        const uniqueKey = getUniqueKey(fn);
        functionMap.set(uniqueKey, { ...fn, uniqueKey, children: [] });
      });
      
      // Build the tree structure
      functions.forEach(fn => {
        const parentKey = getUniqueKey(fn);
        const parentNode = functionMap.get(parentKey);
        
        if (fn.called) {
          fn.called.forEach(calledFn => {
            const childKey = getUniqueKey(calledFn);
            let childNode = functionMap.get(childKey);
            
            // If child node doesn't exist in main functions, create it as a leaf
            if (!childNode) {
              childNode = { ...calledFn, uniqueKey: childKey, children: [] };
            }
            
            parentNode.children.push(childNode);
          });
        }
      });
      
      // Find root nodes (functions that are not called by others)
      functions.forEach(fn => {
        const fnKey = getUniqueKey(fn);
        const isRoot = !functions.some(f => 
          f.called && f.called.some(c => getUniqueKey(c) === fnKey)
        );
        
        if (isRoot) {
          rootNodes.push(functionMap.get(fnKey));
        }
      });
      
      return rootNodes;
    };
    
    return buildTree(data);
  }, [data]);

  // Initialize with better default position
  useEffect(() => {
    if (treeData.length > 0 && pan.x === 0 && pan.y === 0) {
      // Set initial pan to center the content better
      setPan({ x: 200, y: 300 });
      setZoom(0.7);
    }
  }, [treeData, pan.x, pan.y]);

  const toggleNode = useCallback((nodeKey) => {
    setExpandedNodes(prev => {
      const newExpanded = new Set(prev);
      if (newExpanded.has(nodeKey)) {
        newExpanded.delete(nodeKey);
      } else {
        newExpanded.add(nodeKey);
      }
      return newExpanded;
    });
  }, []);

  const collapseAll = useCallback(() => {
    setExpandedNodes(new Set());
    // Also reset view to show all root nodes clearly
    setPan({ x: 200, y: 300 });
    setZoom(0.7);
  }, []);

  const handleMouseDown = useCallback((e) => {
    // Allow dragging on container or SVG background, but not on nodes
    if (e.target.classList.contains('mind-map-container') || 
        e.target.classList.contains('mind-map-svg') ||
        e.target.tagName === 'svg') {
      e.preventDefault();
      setIsDragging(true);
      setLastMousePosition({ x: e.clientX, y: e.clientY });
    }
  }, []);

  const handleMouseMove = useCallback((e) => {
    // Prevent default to avoid text selection while dragging
    if (isDragging) {
      e.preventDefault();
    }
  }, [isDragging]);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  const handleWheel = useCallback((e) => {
    e.preventDefault();
    
    const container = e.currentTarget;
    const rect = container.getBoundingClientRect();
    
    // Mouse position relative to the container
    const mouseX = e.clientX - rect.left;
    const mouseY = e.clientY - rect.top;
    
    // Zoom factor
    const zoomFactor = e.deltaY > 0 ? 0.9 : 1.1;
    const newZoom = Math.max(0.1, Math.min(5, zoom * zoomFactor));
    
    if (newZoom !== zoom) {
      // Calculate the point to zoom into
      const zoomPointX = (mouseX - pan.x) / zoom;
      const zoomPointY = (mouseY - pan.y) / zoom;
      
      // Calculate new pan to keep the zoom point under the cursor
      const newPanX = mouseX - zoomPointX * newZoom;
      const newPanY = mouseY - zoomPointY * newZoom;
      
      setZoom(newZoom);
      setPan({ x: newPanX, y: newPanY });
    }
  }, [zoom, pan]);

  // Set up wheel event listener with non-passive option to allow preventDefault
  useEffect(() => {
    const containerElement = containerRef.current;
    if (!containerElement) return;

    const handleWheelNonPassive = (e) => handleWheel(e);

    // Add wheel event listener with passive: false to allow preventDefault
    containerElement.addEventListener('wheel', handleWheelNonPassive, { passive: false });

    // Cleanup function
    return () => {
      containerElement.removeEventListener('wheel', handleWheelNonPassive);
    };
  }, [handleWheel]);

  const calculateSubtreeHeight = (nodes, level) => {
    if (!nodes || nodes.length === 0) return 0;
    
    let totalHeight = 0;
    nodes.forEach(node => {
      totalHeight += 100; // Increased base spacing for each node
      const nodeKey = node.uniqueKey || `${node.name}@${node.filePath}`;
      if (expandedNodes.has(nodeKey) && node.children && node.children.length > 0) {
        totalHeight += calculateSubtreeHeight(node.children, level + 1);
      }
    });
    
    // Add extra spacing for deeper levels to prevent overlap
    return totalHeight + (level * 20);
  };

  const renderTree = (nodes, level = 0, parentX = 0, parentY = 0, startY = 0) => {
    let currentY = startY;
    
    return nodes.map((node, index) => {
      const nodeWidth = Math.max(280, 200 + (node.name.length * 6)); // Better width calculation
      const x = level === 0 ? 50 : parentX + nodeWidth + 180; // More horizontal spacing
      const y = currentY;
      const nodeKey = node.uniqueKey || `${node.name}@${node.filePath}`;
      const isExpanded = expandedNodes.has(nodeKey);
      const hasChildren = node.children && node.children.length > 0;
      
      // Calculate proper spacing for next node
      let nextY = y + 100; // Increased base spacing
      if (isExpanded && hasChildren) {
        const subtreeHeight = calculateSubtreeHeight(node.children, level + 1);
        nextY = y + Math.max(120, subtreeHeight + 40);
      }
      
      const result = (
        <g key={nodeKey}>
          {/* Curved connection to parent */}
          {level > 0 && (
            <path
              className="connection-line"
              d={`M ${parentX + nodeWidth} ${parentY}
                 C ${parentX + nodeWidth + 75} ${parentY},
                   ${x - 75} ${y},
                   ${x} ${y}`}
              fill="none"
              stroke="#6b7280"
              strokeWidth="2"
            />
          )}
          
          {/* Node */}
          <Node
            node={node}
            x={x}
            y={y}
            width={nodeWidth}
            isExpanded={isExpanded}
            hasChildren={hasChildren}
            isSelected={selectedNode && selectedNode.uniqueKey === nodeKey}
            onToggle={() => toggleNode(nodeKey)}
            onSelect={() => onNodeSelect(node)}
            level={level}
          />
          
          {/* Render children if expanded */}
          {isExpanded && hasChildren && 
            renderTree(node.children, level + 1, x, y, y - ((node.children.length - 1) * 50))
          }
        </g>
      );
      
      currentY = nextY;
      return result;
    });
  };

  return (
    <div 
      ref={containerRef}
      className={`mind-map-container ${isDragging ? 'dragging' : ''}`}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
    >
      <svg
        className="mind-map-svg"
        width="100%"
        height="100%"
        viewBox="-2000 -2000 8000 8000"
        preserveAspectRatio="xMidYMid meet"
        onMouseDown={handleMouseDown}
        style={{
          transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
          transformOrigin: '0 0'
        }}
      >
        <defs>
          <filter id="glow">
            <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
            <feMerge> 
              <feMergeNode in="coloredBlur"/>
              <feMergeNode in="SourceGraphic"/> 
            </feMerge>
          </filter>
        </defs>
        
        {treeData.length > 0 && renderTree(treeData, 0, 0, 0, 0)}
      </svg>
      
      <div className="controls">
        <button onClick={() => setZoom(prev => Math.min(5, prev * 1.2))} className="zoom-btn">
          +
        </button>
        <button onClick={() => setZoom(prev => Math.max(0.1, prev * 0.8))} className="zoom-btn">
          -
        </button>
        <button onClick={collapseAll} className="collapse-btn">
          Collapse All
        </button>
        <button onClick={() => { setPan({ x: 200, y: 300 }); setZoom(0.7); }} className="reset-btn">
          Reset
        </button>
      </div>
    </div>
  );
};

export default MindMap;