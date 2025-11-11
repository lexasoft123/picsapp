import React, { useState, useEffect, useRef } from 'react';
import './Presentation.css';

function Presentation() {
  const [pictures, setPictures] = useState([]);
  const [loading, setLoading] = useState(true);
  const wsRef = useRef(null);
  const prevPositionsRef = useRef(new Map());
  const [swappingIds, setSwappingIds] = useState(new Set());
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const isInitialLoadRef = useRef(true);
  const [layout, setLayout] = useState('grid'); // 'grid' | 'spiral'
  const containerRef = useRef(null);
  const [containerSize, setContainerSize] = useState({ width: 0, height: 0 });
  const [debugSpiral, setDebugSpiral] = useState(false);
  // Physics for spiral layout
  const positionsRef = useRef([]); // [{x,y,size}]
  const targetsRef = useRef([]); // [{x,y,size}]
  const animRef = useRef(null);
  const [tick, setTick] = useState(0); // force re-render
  const lastIdsKeyRef = useRef('');

  useEffect(() => {
    let ws = null;
    let isMounted = true;
    let reconnectTimeout = null;

    // Initial fetch
    fetch('/api/presentation')
      .then((res) => {
        if (!res.ok) {
          throw new Error('Failed to fetch presentation');
        }
        return res.json();
      })
      .then((data) => {
        if (isMounted) {
          const newPictures = Array.isArray(data) ? data : [];
          // Initialize previous positions
          const positions = new Map();
          newPictures.forEach((pic, index) => {
            positions.set(pic.id, index);
          });
          prevPositionsRef.current = positions;
          setPictures(newPictures);
          setLoading(false);
          isInitialLoadRef.current = false;
          setIsInitialLoad(false);
        }
      })
      .catch((err) => {
        console.error('Error fetching presentation:', err);
        if (isMounted) {
          setPictures([]);
          setLoading(false);
        }
      });

    // WebSocket connection
    // In development, connect directly to the Go server on port 8080
    // In production, use the same host
    const isDev = window.location.hostname === 'localhost' && window.location.port === '3000';
    const wsHost = isDev ? 'localhost:8080' : window.location.host;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${wsHost}/ws`;

    const connectWebSocket = () => {
      if (!isMounted) return;

      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket connected');
        if (reconnectTimeout) {
          clearTimeout(reconnectTimeout);
          reconnectTimeout = null;
        }
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (isMounted) {
            const newPictures = Array.isArray(data) ? data : [];
            
            // Detect position changes
            const newPositions = new Map();
            const changedIds = new Set();
            
            newPictures.forEach((pic, index) => {
              newPositions.set(pic.id, index);
              const prevIndex = prevPositionsRef.current.get(pic.id);
              if (prevIndex !== undefined && prevIndex !== index) {
                changedIds.add(pic.id);
              }
            });
            
            // Also check if any picture moved from a different position
            prevPositionsRef.current.forEach((prevIndex, id) => {
              const newIndex = newPositions.get(id);
              if (newIndex !== undefined && newIndex !== prevIndex) {
                changedIds.add(id);
              }
            });
            
            // Update swapping state
            if (changedIds.size > 0 && !isInitialLoadRef.current) {
              setSwappingIds(new Set(changedIds));
              // Clear swapping state after animation
              setTimeout(() => {
                if (isMounted) {
                  setSwappingIds(new Set());
                }
              }, 600);
            }
            
            // Update previous positions
            prevPositionsRef.current = newPositions;
            setPictures(newPictures);
            if (isInitialLoadRef.current) {
              isInitialLoadRef.current = false;
              setIsInitialLoad(false);
            }
          }
        } catch (err) {
          console.error('Error parsing WebSocket message:', err);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      ws.onclose = (event) => {
        if (!isMounted) return;

        if (event.wasClean) {
          console.log('WebSocket disconnected cleanly');
        } else {
          console.log('WebSocket connection lost, attempting to reconnect...');
          // Attempt to reconnect after a delay
          reconnectTimeout = setTimeout(() => {
            if (isMounted) {
              connectWebSocket();
            }
          }, 3000);
        }
      };

      wsRef.current = ws;
    };

    connectWebSocket();

    return () => {
      isMounted = false;
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
      if (wsRef.current) {
        if (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING) {
          wsRef.current.close();
        }
      }
    };
  }, []);

  // Measure container for spiral positioning (must be before any early returns)
  useEffect(() => {
    const measure = () => {
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect();
        setContainerSize({ width: rect.width, height: rect.height });
      }
    };
    measure();
    window.addEventListener('resize', measure);
    return () => window.removeEventListener('resize', measure);
  }, []);

  // Compute derived arrays for hooks
  const safePictures = pictures || [];
  const limitedPictures = safePictures.slice(0, 30);
  const idsKey = limitedPictures.map((p) => p.id).join('|');

  // Update targets whenever data or size changes
  useEffect(() => {
    if (layout !== 'spiral') return;
    const w = containerSize.width || 1200;
    const h = containerSize.height || 800;
    const cx = w / 2;
    const cy = h / 2 + 100;
    const n = Math.max(1, limitedPictures.length);
    const scale = Math.max(0.65, Math.min(1.6, 30 / n));
    const baseMax = Math.min(360, Math.max(220, Math.min(w, h) * 0.28));
    // First picture max size, clamp to 60% viewport and baseMax * 2
    const maxSize = Math.min(Math.min(w, h) * 0.6, baseMax * 2);
    const minSize = Math.max(64, Math.floor(maxSize / 2)); // last picture exactly half of first

    // Compute spiral bounds now that sizes are known
    const margin = 8;
    // Use Archimedean spiral: r = a + b * theta
    // Set maxR explicitly to min(page_height, page_width)/2
    const maxR = Math.min(w, h) / 2;
    const a = Math.max(0, Math.min(80, minSize * 0.6));
    const thetaStep = Math.PI / 4;
    const thetaLast = (n > 1) ? (n - 1) * thetaStep : thetaStep;
    const b = thetaLast > 0 ? (maxR - a) / thetaLast : 0;
    const skipSteps = 7; // number of points to skip after top picture

    // Size mapping based on likes with 2x ratio from top to last
    const maxLikes = n > 0 ? (limitedPictures[0]?.likes ?? 0) : 0;
    const minLikes = n > 0 ? (limitedPictures[n - 1]?.likes ?? 0) : 0;
    const likesRange = Math.max(0, maxLikes - minLikes);
    const newTargets = [];
    for (let i = 0; i < limitedPictures.length; i++) {
      const theta = (i === 0 ? 0 : (i + skipSteps - 1) * thetaStep);
      // Archimedean radius
      const r = (i === 0 ? 0 : a + b * theta);
      const x = cx + r * Math.cos(theta);
      const y = cy + r * Math.sin(theta);
      // Size interpolation: first maxSize, last minSize
      let size;
      if (n === 1) {
        size = maxSize;
      } else {
        const t = i / (n - 1); // 0..1
        size = Math.round(maxSize - t * (maxSize - minSize));
      }
      newTargets.push({ x, y, size });
    }
    targetsRef.current = newTargets;

    const pos = positionsRef.current;
    const orderChanged = lastIdsKeyRef.current !== idsKey;
    if (!pos || pos.length !== newTargets.length) {
      positionsRef.current = newTargets.map((t) => ({ x: t.x, y: t.y, size: t.size }));
    } else {
      for (let i = 0; i < newTargets.length; i++) {
        pos[i].size = newTargets[i].size;
      }
      if (orderChanged) {
        // Keep current positions but ensure subsequent interpolation starts fresh
        // (no additional action needed beyond updating targets)
      }
    }
    lastIdsKeyRef.current = idsKey;
    setTick((t) => t + 1);
  }, [layout, containerSize.width, containerSize.height, limitedPictures.length, idsKey]);

  // Physics animation loop
  useEffect(() => {
    if (layout !== 'spiral') {
      if (animRef.current) cancelAnimationFrame(animRef.current);
      return;
    }
    let last = performance.now();
    const baseSmoothing = 0.14;

    const step = (now) => {
      const dt = Math.min(0.05, (now - last) / 1000);
      last = now;
      const targets = targetsRef.current;
      const pos = positionsRef.current;
      if (!targets || !pos) {
        animRef.current = requestAnimationFrame(step);
        return;
      }

      const alpha = Math.min(1, baseSmoothing * (dt * 60));

      for (let i = 0; i < pos.length; i++) {
        pos[i].x += (targets[i].x - pos[i].x) * alpha;
        pos[i].y += (targets[i].y - pos[i].y) * alpha;
        pos[i].size += (targets[i].size - pos[i].size) * alpha;
      }
      positionsRef.current = [...pos];
      setTick((t) => (t + 1) % 1000000);
      animRef.current = requestAnimationFrame(step);
    };
    animRef.current = requestAnimationFrame(step);
    return () => {
      if (animRef.current) cancelAnimationFrame(animRef.current);
    };
  }, [layout]);

  if (loading) {
    return (
      <div className="presentation-page">
        <div className="loading">Loading presentation...</div>
      </div>
    );
  }

  
  const renderSpiral = () => {
    const w = containerSize.width || 1200;
    const h = containerSize.height || 800;
    const cx = w / 2;
    const cy = h / 2;
    const items = [];

    // Use physics-driven positions
    const pos = positionsRef.current;
    if (debugSpiral) {
      // Render dots at target centers with likes labels
      const targets = targetsRef.current && targetsRef.current.length === limitedPictures.length ? targetsRef.current : [];
      for (let i = 0; i < limitedPictures.length; i++) {
        const t = targets[i] || pos[i] || { x: cx, y: cy };
        const showCount = i < 10;
        const likeCount = limitedPictures[i]?.likes ?? 0;
        items.push(
          <div key={`dot-${i}`} className="spiral-dot" style={{ left: t.x, top: t.y }}>
            {showCount && (
              <div className="spiral-label">
                <span className="spiral-like-icon">❤️</span>
                <span>{likeCount}</span>
              </div>
            )}
          </div>
        );
      }
    } else {
      for (let i = 0; i < limitedPictures.length; i++) {
        const pic = limitedPictures[i];
        const p = pos[i] || { x: cx, y: cy, size: 200 };
        const classNames = i === 0 ? `spiral-card top ${swappingIds.has(pic.id) ? 'swapping' : ''}` : `spiral-card ${swappingIds.has(pic.id) ? 'swapping' : ''}`;
        const showCount = i < 10;
        const likeCount = pic.likes;
        items.push(
          <div
            key={pic.id}
            className={classNames}
            style={{ left: p.x, top: p.y, width: p.size, height: p.size, zIndex: (limitedPictures.length - i) }}
          >
            {showCount && (
              <div className="spiral-like-tag">
                <span className="spiral-like-icon">❤️</span>
                <span className="spiral-like-count">{likeCount}</span>
              </div>
            )}
            <img src={pic.url} alt={pic.filename} className="spiral-image" />
            <div className="card-info">
              <div className="card-likes">
                <span className="likes-icon">❤️</span>
                <span className="likes-count">{pic.likes}</span>
              </div>
            </div>
          </div>
        );
      }
    }

    return (
      <div ref={containerRef} className="presentation-spiral">
        {items}
      </div>
    );
  };

  return (
    <div className="presentation-page">
      <div className="container">
        <div className="presentation-header">
          <h1 className="page-title">Top Pictures by Likes</h1>
          <div className="layout-switch">
            <button
              className={`layout-btn ${layout === 'grid' ? 'active' : ''}`}
              onClick={() => setLayout('grid')}
              aria-label="Grid layout"
            >
              Grid
            </button>
            <button
              className={`layout-btn ${layout === 'spiral' ? 'active' : ''}`}
              onClick={() => setLayout('spiral')}
              aria-label="Spiral layout"
            >
              Spiral
            </button>
            {layout === 'spiral' && (
              <button
                className={`layout-btn ${debugSpiral ? 'active' : ''}`}
                onClick={() => setDebugSpiral((v) => !v)}
                aria-label="Toggle spiral debug"
              >
                Debug
              </button>
            )}
          </div>
        </div>

        {limitedPictures.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">🖼️</div>
            <div className="empty-text">No pictures yet.</div>
          </div>
        ) : layout === 'grid' ? (
          <div className="presentation-grid">
            {limitedPictures.map((picture, index) => {
              const isSwapping = swappingIds.has(picture.id);
              return (
                <div
                  key={picture.id}
                  className={`presentation-card ${isSwapping ? 'swapping' : ''} ${isInitialLoad ? 'initial' : ''}`}
                  style={{ animationDelay: isInitialLoad ? `${index * 0.1}s` : '0s' }}
                >
                  <div className="card-rank">#{index + 1}</div>
                  <img src={picture.url} alt={picture.filename} className="presentation-image" />
                  <div className="card-info">
                    <div className="card-likes">
                      <span className="likes-icon">❤️</span>
                      <span className="likes-count">{picture.likes}</span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          renderSpiral()
        )}
      </div>
    </div>
  );
}

export default Presentation;

