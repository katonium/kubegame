import type { SVGProps } from 'react';

export function StrawberryIcon(props: SVGProps<SVGSVGElement>) {
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
      <path
        d="M12 2C8.686 2 6 4.686 6 8c0 4 6 12 6 12s6-8 6-12c0-3.314-2.686-6-6-6z"
        fill="#FF6347"
        stroke="#FF4500"
      />
    </svg>
  );
}
