import { useContext } from 'react';

import { Context, PluginContextType } from './PluginContext';

export function usePluginContext(): PluginContextType | null {
  const context = useContext(Context);

  // The extensions hooks (e.g. `usePluginLinks()`) are using this hook to check
  // if they are inside a plugin or not (core Grafana).
  if (!context) {
    return null;
  }

  return context;
}
