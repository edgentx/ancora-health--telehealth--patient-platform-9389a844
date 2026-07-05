import type { Metadata } from 'next';
import { Plus_Jakarta_Sans } from 'next/font/google';

import { Providers } from '@/components/providers';
import { resolveIdentity } from '@/lib/identity';

import './globals.css';

// Design-system typography baseline (design-system/MASTER.md).
const jakarta = Plus_Jakarta_Sans({
  subsets: ['latin'],
  weight: ['400', '600', '700', '800'],
  variable: '--font-sans',
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'Ancora Health',
  description: 'Telehealth & Patient Platform',
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  // Resolve identity once at the root from the edge's trusted headers, then hand
  // it to the client providers to hydrate the store. No token parsing here.
  const identity = await resolveIdentity();

  return (
    <html lang="en" className={jakarta.variable}>
      <body>
        <Providers identity={identity}>{children}</Providers>
      </body>
    </html>
  );
}
