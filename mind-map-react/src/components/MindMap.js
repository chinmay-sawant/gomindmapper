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

  const renderTree = (nodes, level = 0, parentX = 400, parentY = 100) => {
    return nodes.map((node, index) => {
      const x = level === 0 ? parentX : parentX + 300;
      const y = level === 0 ? parentY + (index * 120) : parentY + (index * 80) - ((nodes.length - 1) * 40);
      const isExpanded = expandedNodes.has(node.name);
      const hasChildren = node.children && node.children.length > 0;
      
      return (
        <g key={`${node.name}-${level}-${index}`}>
          {/* Connection line to parent */}
          {level > 0 && (
            <line
              x1={parentX + 140}
              y1={parentY}
              x2={x - 10}
              y2={y}
              className="connection-line"
            />
          )}
          
          {/* Node */}
          <Node
            node={node}
            x={x}
            y={y}
            isExpanded={isExpanded}
            hasChildren={hasChildren}
            isSelected={selectedNode && selectedNode.name === node.name}
            onToggle={() => toggleNode(node.name)}
            onSelect={() => onNodeSelect(node)}
            level={level}
          />
          
          {/* Render children if expanded */}
          {isExpanded && hasChildren && 
            renderTree(node.children, level + 1, x, y)
          }
        </g>
      );
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
        
        {treeData.length > 0 && renderTree(treeData)}
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