import React, { useState, useMemo, useCallback } from 'react';
import Node from './Node';
import './MindMap.css';

const MindMap = ({ data, selectedNode, onNodeSelect }) => {
  const [expandedNodes, setExpandedNodes] = useState(new Set());
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [lastMousePosition, setLastMousePosition] = useState({ x: 0, y: 0 });

  // Build the tree structure from flat function data
  const treeData = useMemo(() => {
    const buildTree = (functions) => {
      const functionMap = new Map();
      const rootNodes = [];
      
      // Create a map of all functions
      functions.forEach(fn => {
        functionMap.set(fn.name, { ...fn, children: [] });
      });
      
      // Build the tree structure
      functions.forEach(fn => {
        const node = functionMap.get(fn.name);
        if (fn.called) {
          fn.called.forEach(calledFn => {
            const childNode = functionMap.get(calledFn.name) || { ...calledFn, children: [] };
            node.children.push(childNode);
          });
        }
        
        // Check if this is a root node (not called by others)
        const isRoot = !functions.some(f => 
          f.called && f.called.some(c => c.name === fn.name)
        );
        
        if (isRoot) {
          rootNodes.push(node);
        }
      });
      
      return rootNodes;
    };
    
    return buildTree(data);
  }, [data]);

  const toggleNode = useCallback((nodeName) => {
    setExpandedNodes(prev => {
      const newExpanded = new Set(prev);
      if (newExpanded.has(nodeName)) {
        newExpanded.delete(nodeName);
      } else {
        newExpanded.add(nodeName);
      }
      return newExpanded;
    });
  }, []);

  const handleMouseDown = useCallback((e) => {
    if (e.target.classList.contains('mind-map-container')) {
      setIsDragging(true);
      setLastMousePosition({ x: e.clientX, y: e.clientY });
    }
  }, []);

  const handleMouseMove = useCallback((e) => {
    if (isDragging) {
      const deltaX = e.clientX - lastMousePosition.x;
      const deltaY = e.clientY - lastMousePosition.y;
      
      setPan(prev => ({
        x: prev.x + deltaX,
        y: prev.y + deltaY
      }));
      
      setLastMousePosition({ x: e.clientX, y: e.clientY });
    }
  }, [isDragging, lastMousePosition]);

  const handleMouseUp = useCallback(() => {
    setIsDragging(false);
  }, []);

  const handleWheel = useCallback((e) => {
    e.preventDefault();
    const zoomFactor = e.deltaY > 0 ? 0.9 : 1.1;
    setZoom(prev => Math.max(0.3, Math.min(3, prev * zoomFactor)));
  }, []);

  const calculateSubtreeHeight = (nodes, level) => {
    let height = 0;
    nodes.forEach(node => {
      height += 60; // Child node spacing
      if (expandedNodes.has(node.name) && node.children && node.children.length > 0) {
        height += calculateSubtreeHeight(node.children, level + 1);
      }
    });
    return height;
  };

  const renderTree = (nodes, level = 0, parentX = 0, parentY = 0, startY = 0) => {
    let currentY = startY;
    
    return nodes.map((node, index) => {
      const nodeWidth = 200 + (node.name.length * 8); // Dynamic width based on text
      const x = level === 0 ? 50 : parentX + 320;
      const y = currentY;
      const isExpanded = expandedNodes.has(node.name);
      const hasChildren = node.children && node.children.length > 0;
      
      // Calculate spacing for next node
      let nextY = y + 80; // Base spacing
      if (isExpanded && hasChildren) {
        nextY = y + calculateSubtreeHeight(node.children, level + 1) + 60;
      }
      
      const result = (
        <g key={`${node.name}-${level}-${index}`}>
          {/* Curved connection to parent */}
          {level > 0 && (
            <path
              className="connection-line"
              d={`M ${parentX + 140} ${parentY}
                 C ${parentX + 200} ${parentY},
                   ${x - 60} ${y},
                   ${x - 18} ${y}`}
              fill="none"
              stroke="#404040"
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
            isSelected={selectedNode && selectedNode.name === node.name}
            onToggle={() => toggleNode(node.name)}
            onSelect={() => onNodeSelect(node)}
            level={level}
          />
          
          {/* Render children if expanded */}
          {isExpanded && hasChildren && 
            renderTree(node.children, level + 1, x, y, y - ((node.children.length - 1) * 30))
          }
        </g>
      );
      
      currentY = nextY;
      return result;
    });
  };

  return (
    <div 
      className="mind-map-container"
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
      onWheel={handleWheel}
    >
      <svg
        className="mind-map-svg"
        style={{
          transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
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
        
        {treeData.length > 0 && renderTree(treeData, 0, 0, 0, 100)}
      </svg>
      
      <div className="controls">
        <button onClick={() => setZoom(prev => Math.min(3, prev * 1.2))} className="zoom-btn">
          +
        </button>
        <button onClick={() => setZoom(prev => Math.max(0.3, prev * 0.8))} className="zoom-btn">
          -
        </button>
        <button onClick={() => { setPan({ x: 0, y: 0 }); setZoom(1); }} className="reset-btn">
          Reset
        </button>
      </div>
    </div>
  );
};

export default MindMap;