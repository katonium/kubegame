import type { SVGProps } from 'react';

export function BananaIcon(props: SVGProps<SVGSVGElement>) {
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
      <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1" fill="#FFD700" stroke="#DAA520" />
      <path d="M5 16s1 1 4 1 5-2 8-2 4 1 4 1" fill="#F0E68C" stroke="#DAA520" />
    </svg>
  );
}
