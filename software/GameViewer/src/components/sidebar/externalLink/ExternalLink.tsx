import React from 'react';
import './ExternalLink.css';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';

interface ExternalLinksProps {
  text: string;
  link: string;
  title?: string | undefined;
}

const ExternalLinks: React.FC<ExternalLinksProps> = ({
  text,
  link,
  title,
}: ExternalLinksProps) => {
  return (
    <div className="externalLinks-wrapper">
      <a href={link} target="_blank" rel="noopener noreferrer" title={title}>
        <OpenInNewIcon className="icon" /> {text}
      </a>
    </div>
  );
};

export default ExternalLinks;
