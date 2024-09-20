export const APIs = {
    Alerting: {
        eval: '/api/v1/eval',
    },
    DataSource: {
        resourcePattern: '/api/datasources/*/resources',
        resourceUIDPattern: '/api/datasources/uid/*/resources',
        queryPattern: '*/**/api/ds/query*',
        query: '/api/ds/query',
        health: (uid: string, _: string) => `/api/datasources/uid/${uid}/health`,
        datasourceByUID: (uid: string) => `/api/datasources/uid/${uid}`,
        proxy: (uid: string, _: string) => `api/datasources/proxy/uid/${uid}`,
    },
    Dashboard: {
        delete: (uid: string) => `/api/dashboards/uid/${uid}`,
    },
    Plugin: {
        settings: (pluginId: string) => `/api/plugins/${pluginId}/settings`,
    },
};
