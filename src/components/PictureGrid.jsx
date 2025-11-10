import React from 'react';
import PictureCard from './PictureCard';
import './PictureGrid.css';

function PictureGrid({ pictures, onLike }) {
  const safePictures = pictures || [];
  
  return (
    <div className="picture-grid">
      {safePictures.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">🖼️</div>
          <div className="empty-text">No pictures yet. Upload one to get started!</div>
        </div>
      ) : (
        safePictures.map((picture) => (
          <PictureCard key={picture.id} picture={picture} onLike={onLike} />
        ))
      )}
    </div>
  );
}

export default PictureGrid;

