import { useMemo } from 'react';
import { useObservable } from 'react-use';

import { usePluginContext } from '@grafana/data';
import {
  UsePluginComponentOptions,
  UsePluginComponentsResult,
} from '@grafana/runtime/src/services/pluginExtensions/getPluginExtensions';

import { useAddedComponentsRegistry } from './ExtensionRegistriesContext';
import { isExtensionPointIdInvalid, isExtensionPointMetaInfoMissing, isGrafanaDevMode } from './utils';

// Returns an array of component extensions for the given extension point
export function usePluginComponents<Props extends object = {}>({
  limitPerPlugin,
  extensionPointId,
}: UsePluginComponentOptions): UsePluginComponentsResult<Props> {
  const registry = useAddedComponentsRegistry();
  const registryState = useObservable(registry.asObservable());
  const pluginContext = usePluginContext();

  return useMemo(() => {
    const components: Array<React.ComponentType<Props>> = [];
    const extensionsByPlugin: Record<string, number> = {};

    if (isGrafanaDevMode && isExtensionPointIdInvalid(extensionPointId, pluginContext)) {
      console.error(
        `usePluginComponents("${extensionPointId}") - The extension point ID "${extensionPointId}" is invalid.`
      );
      return {
        isLoading: false,
        components: [],
      };
    }

    if (isGrafanaDevMode && isExtensionPointMetaInfoMissing(extensionPointId, pluginContext)) {
      console.error(
        `usePluginComponents("${extensionPointId}") - The extension point is missing from the "plugin.json" file.`
      );
      return {
        isLoading: false,
        components: [],
      };
    }

    for (const registryItem of registryState?.[extensionPointId] ?? []) {
      const { pluginId } = registryItem;

      // Only limit if the `limitPerPlugin` is set
      if (limitPerPlugin && extensionsByPlugin[pluginId] >= limitPerPlugin) {
        continue;
      }

      if (extensionsByPlugin[pluginId] === undefined) {
        extensionsByPlugin[pluginId] = 0;
      }

      components.push(registryItem.component as React.ComponentType<Props>);
      extensionsByPlugin[pluginId] += 1;
    }

    return {
      isLoading: false,
      components,
    };
  }, [extensionPointId, limitPerPlugin, pluginContext, registryState]);
}
