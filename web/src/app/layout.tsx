import type { Metadata } from 'next';
import { Plus_Jakarta_Sans } from 'next/font/google';

import { Providers } from '@/components/providers';
import { publicConfigSnapshot, serializeRuntimeEnvScript } from '@/lib/env';
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

  // Resolve endpoint config from the container environment for this request and
  // seed it into the document. A synchronous inline script in <head> runs before
  // any client bundle, so the browser reads runtime values — the same image
  // retargets to any environment via env vars, no rebuild (S-83).
  const runtimeEnvScript = serializeRuntimeEnvScript(publicConfigSnapshot());

  return (
    <html lang="en" className={jakarta.variable}>
      <head>
        <script dangerouslySetInnerHTML={{ __html: runtimeEnvScript }} />
      </head>
      <body>
        <Providers identity={identity}>{children}</Providers>
      </body>
    </html>
  );
}
