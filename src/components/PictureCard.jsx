import React, { useState } from 'react';
import './PictureCard.css';

function PictureCard({ picture, onLike }) {
  const [liked, setLiked] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);

  const handleLike = (e) => {
    e.stopPropagation();
    if (!liked) {
      setLiked(true);
      onLike(picture.id);
      setTimeout(() => setLiked(false), 1000);
    }
  };

  return (
    <div className="picture-card">
      <div className="picture-wrapper">
        {!imageLoaded && <div className="image-placeholder" />}
        <img
          src={picture.url}
          alt={picture.filename}
          className="picture-image"
          onLoad={() => setImageLoaded(true)}
          style={{ display: imageLoaded ? 'block' : 'none' }}
        />
        <div className="picture-overlay">
          <button
            className={`like-button ${liked ? 'liked' : ''}`}
            onClick={handleLike}
            aria-label="Like picture"
          >
            <span className="like-icon">❤️</span>
            <span className="like-count">{picture.likes}</span>
          </button>
        </div>
      </div>
    </div>
  );
}

export default PictureCard;

