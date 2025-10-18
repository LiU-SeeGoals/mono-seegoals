import React, { useState } from 'react';
import './Header.css';

interface HeaderProps {}

const Header: React.FC<HeaderProps> = () => {
  return (
    <div className="header-wrapper">
      <h1 style={{ paddingRight: '20px' }}>SeeGoals</h1>
      <img src="./src/assets/Fia_logo.png" alt="logo" className="logo" />
    </div>
  );
};

export default Header;
