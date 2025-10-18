import React, { useState } from 'react';
import './ButtonSetting.css';
import Button from '@mui/material/Button';
import ArrowRightIcon from '@mui/icons-material/ArrowRight';
import InfoIcon from '@mui/icons-material/Info';

interface SettingsProps {}

const Settings: React.FC<SettingsProps> = () => {
  const [isDropdownVisible, setDropdownVisible] = useState(false);

  const toggleDropdown = () => {
    setDropdownVisible(!isDropdownVisible);
  };

  return (
    <>
      <div className="buttonSetting-wrapper">
        <p onClick={toggleDropdown}>
          <ArrowRightIcon className="icon-right-arrow" />
          Set robot position
          <span className="icon">
            <InfoIcon className="icon-info" />
            <span className="tooltip">Not yet implemented</span>
          </span>
        </p>
        <Button>Ugglan</Button>
      </div>
      {isDropdownVisible && (
        <div className="button-dropdown-content">
          <div className="buttonSetting-wrapper">
            <p>Alt 1</p>
            <Button>Kråkan</Button>
          </div>
          <div className="buttonSetting-wrapper">
            <p>Alt 2</p>
            <Button>Örnen</Button>
          </div>
          <div className="buttonSetting-wrapper">
            <p>Alt 3</p>
            <Button>Lärkan</Button>
          </div>
          <div className="buttonSetting-wrapper">
            <p>Alt 4</p>
            <Button>Ugglan</Button>
          </div>
        </div>
      )}
    </>
  );
};

export default Settings;
