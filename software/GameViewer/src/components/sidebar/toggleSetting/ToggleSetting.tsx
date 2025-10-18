import React, { useState, useEffect } from 'react';
import './ToggleSetting.css';
import Switch from '@mui/material/Switch';
import ArrowRightIcon from '@mui/icons-material/ArrowRight';
import InfoIcon from '@mui/icons-material/Info';

interface SettingsProps {
  name: string;
  settingsBlue: boolean[];
  settingsYellow: boolean[];
  setSettingsBlue: React.Dispatch<React.SetStateAction<boolean[]>>;
  setSettingsYellow: React.Dispatch<React.SetStateAction<boolean[]>>;
  itemName: string;
  tip: string;
}

const label = { inputProps: { 'aria-label': 'Switch demo' } };

const Settings: React.FC<SettingsProps> = ({
  name,
  settingsBlue,
  setSettingsBlue,
  settingsYellow,
  setSettingsYellow,
  itemName,
  tip,
}) => {
  const [isDropdownVisible, setDropdownVisible] = useState(false);
  const [topSwitch, setTopSwitch] = useState(false);

  const toggleDropdown = () => {
    setDropdownVisible(!isDropdownVisible);
  };

  const toggleSettingBlue = (index: number) => {
    const newSettings = [...settingsBlue];
    newSettings[index] = !newSettings[index];
    setSettingsBlue(newSettings);
  };
  const toggleSettingYellow = (index: number) => {
    const newSettings = [...settingsYellow];
    newSettings[index] = !newSettings[index];
    setSettingsYellow(newSettings);
  };

  const toggleAllSettingsBlue = (value: boolean) => {
    setSettingsBlue(settingsBlue.map(() => value));
  };
  const toggleAllSettingsYellow = (value: boolean) => {
    setSettingsYellow(settingsYellow.map(() => value));
  };

  const handleTopSwitchChange = () => {
    const newValue = !topSwitch;
    setTopSwitch(newValue);
    toggleAllSettingsBlue(newValue);
    toggleAllSettingsYellow(newValue);
  };

  return (
    <>
      <div className="toggleSetting-wrapper">
        <p onClick={toggleDropdown}>
          <ArrowRightIcon className="icon-right-arrow" />
          {name}
          <span className="icon">
            <InfoIcon className="icon-info" />
            <span className="tooltip">{tip}</span>
          </span>
        </p>
        <Switch
          {...label}
          checked={topSwitch}
          onChange={handleTopSwitchChange}
        />
      </div>
      {isDropdownVisible && (
        <div className="row">
          <div className="toggle-dropdown-content">
            <h3 className="test">Blue robots</h3>

            {settingsBlue.map((setting, index) => (
              <div className="toggleSetting-wrapper" key={index}>
                <p>{`${itemName} ${index}`}</p>
                <Switch
                  {...label}
                  checked={setting}
                  onChange={() => toggleSettingBlue(index)}
                />
              </div>
            ))}
          </div>

          <div className="toggle-dropdown-content">
            <h3 className="test">Yellow robots</h3>
            {settingsYellow.map((setting, index) => (
              <div className="toggleSetting-wrapper" key={index}>
                <p>{`${itemName} ${index}`}</p>
                <Switch
                  {...label}
                  checked={setting}
                  onChange={() => toggleSettingYellow(index)}
                />
              </div>
            ))}
          </div>
        </div>
      )}
    </>
  );
};

export default Settings;
