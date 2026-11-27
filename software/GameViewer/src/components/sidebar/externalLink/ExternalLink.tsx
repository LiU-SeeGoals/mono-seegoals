import React from 'react';
import './ExternalLink.css';

interface ExternalLinkProps {
  text: string;
  link: string;
  title?: string;
}

const ExternalLink: React.FC<ExternalLinkProps> = ({ text, link, title }) => (
  <div className="externalLinks-wrapper">
    <a href={link} target="_blank" rel="noopener noreferrer" title={title}>
      <span className="icon" aria-hidden="true">↗</span> {text}
    </a>
  </div>
);

export default ExternalLink;
