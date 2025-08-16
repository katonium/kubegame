import type { SVGProps } from 'react';

export function ChocolateIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <rect x="5" y="5" width="14" height="14" rx="2" fill="#8B4513" stroke="#A0522D" />
    </svg>
  );
}
