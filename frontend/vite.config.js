/*
Copyright 2023-2026 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import yaml from '@rollup/plugin-yaml';

const customAPIDomain = process.env.API_DOMAIN;

const proxyConfig = {
  changeOrigin: !!customAPIDomain,
  target: customAPIDomain || 'http://localhost:8080',
  configure: (proxy, _options) => {
    proxy.on('error', (err, _req, res) => {
      // Suppress unhandled ECONNREFUSED in Node when backend server is offline
      if (res && !res.headersSent && typeof res.writeHead === 'function') {
        res.writeHead(503, {
          'Content-Type': 'application/json'
        });
        res.end(
          JSON.stringify({
            code: 50300,
            message:
              'DevOps Backend service offline (http://localhost:8080 is unreachable)',
            details: err.message
          })
        );
      }
    });
    proxy.on('proxyReqWs', (_proxyReq, _req, socket) => {
      socket.on('error', () => {
        // Suppress websocket unhandled socket errors when backend is offline
      });
    });
  }
};

export default defineConfig(({ mode }) => ({
  root: './',
  base: './',
  build: {
    // Relative to outDir
    assetsDir: '.',
    // Relative to the root
    outDir: 'dist',
    target: 'es2022'
  },
  css: {
    devSourcemap: true,
    preprocessorOptions: {
      scss: {
        quietDeps: true
      }
    }
  },
  plugins: [react({ devTarget: 'es2022' }), yaml()],
  resolve: {
    extensions: ['.js', '.jsx']
  },
  server: {
    headers: {
      // https://github.com/codemirror/codemirror5/issues/6707
      // style-src blob: 'nonce-tkn-dev';
      'Content-Security-Policy':
        "default-src 'none'; style-src 'self' 'unsafe-inline'; img-src 'self'; script-src 'self' 'unsafe-eval' 'unsafe-inline'; connect-src 'self' wss: ws:; font-src 'self' https://1.www.s81c.com;"
    },
    open: mode !== 'test',
    port: process.env.PORT || 8000,
    proxy: {
      '/v1': proxyConfig,
      '/api': {
        ...proxyConfig,
        ws: true
      },
      '/apis': {
        ...proxyConfig,
        ws: true
      }
    },
    strictPort: mode !== 'test',
    watch: {
      ignored: ['**/coverage/**', '**/storybook-static/**']
    }
  },
  test: {
    coverage: {
      all: false,
      clean: true,
      enabled: true,
      exclude: ['**/src/mock/**'],
      provider: 'istanbul',
      reporter: ['text', 'html'],
      thresholds: {
        branches: 90,
        functions: 90,
        lines: 90,
        statements: 90
      }
    },
    environment: 'jsdom',
    globals: true,
    restoreMocks: true,
    // slowTestThreshold: <num> // millis
    server: {
      deps: {
        inline: ['@uiw/react-codemirror']
      }
    },
    setupFiles: './config_frontend/setupTests.js'
  }
}));
