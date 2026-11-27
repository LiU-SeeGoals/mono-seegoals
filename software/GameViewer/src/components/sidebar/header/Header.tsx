import React from 'react';
import './Header.css';
import logo from '../../../assets/Fia_logo.png';

const Header: React.FC = () => (
  <div className="header-wrapper">
    <h1 style={{ paddingRight: '20px' }}>SeeGoals</h1>
    <img src={logo} alt="logo" className="logo" />
  </div>
);

export default Header;
