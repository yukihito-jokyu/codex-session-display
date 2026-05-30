import React from 'react';
import styles from './TitleBar.module.css';
import { WindowMinimize, WindowToggleMaximize, WindowClose } from '../../../wailsjs/runtime/runtime';

export const TitleBar: React.FC = () => {
  return (
    <header className={styles.titleBar}>
      <div className={styles.dragArea}>
        <span className={styles.title}>Codex Session Display</span>
      </div>
      <div className={styles.controls}>
        <button 
          className={styles.controlBtn} 
          onClick={WindowMinimize} 
          title="Minimize"
          aria-label="Minimize"
        >
          <svg width="10" height="1" viewBox="0 0 10 1">
            <line x1="0" y1="0.5" x2="10" y2="0.5" stroke="currentColor" strokeWidth="1" />
          </svg>
        </button>
        <button 
          className={styles.controlBtn} 
          onClick={WindowToggleMaximize} 
          title="Maximize"
          aria-label="Maximize"
        >
          <svg width="10" height="10" viewBox="0 0 10 10">
            <rect x="0.5" y="0.5" width="9" height="9" fill="none" stroke="currentColor" strokeWidth="1" />
          </svg>
        </button>
        <button 
          className={`${styles.controlBtn} ${styles.closeBtn}`} 
          onClick={WindowClose} 
          title="Close"
          aria-label="Close"
        >
          <svg width="10" height="10" viewBox="0 0 10 10">
            <path d="M 0,0 L 10,10 M 10,0 L 0,10" stroke="currentColor" strokeWidth="1" />
          </svg>
        </button>
      </div>
    </header>
  );
};
