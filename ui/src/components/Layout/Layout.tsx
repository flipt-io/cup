import * as React from 'react';
import '@/App.css';
import { ThemeProvider } from '@/components/theme-provider';
import { ModeToggle } from '@/components/mode-toggle';
import logoLight from '@/assets/cup_light.svg';
import logoDark from '@/assets/cup_dark.svg';

interface Props {
  children: React.ReactNode;
}

const Layout: React.FunctionComponent<Props> = ({ children }) => {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <div className="App">
        <header className="border-b mb-4">
          <div className="max-w-screen-xl flex flex-wrap items-center justify-between mx-auto p-4">
            <a href="/" className="flex items-center">
              <picture>
                <source
                  srcSet={logoDark}
                  media="(prefers-color-scheme: dark)"
                />
                <img src={logoLight} className="h-12 mr-3" alt="Cup Logo" />
              </picture>
              <span className="self-center text-2xl font-semibold whitespace-nowrap">
                Cup
              </span>
            </a>
            <ModeToggle />
          </div>
        </header>
      </div>
      <main className="max-w-screen-xl flex flex-wrap items-center justify-between mx-auto">
        {children}
      </main>
    </ThemeProvider>
  );
};

export default Layout;
