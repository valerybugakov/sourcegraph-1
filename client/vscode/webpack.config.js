// @ts-check

'use strict'
const path = require('path')

const MiniCssExtractPlugin = require('mini-css-extract-plugin')

/** @type {import('webpack').Configuration}*/
function getExtensionConfig(targetType) {
  return {
    name: `extension:${targetType}`,
    target: targetType, // vscode extensions run in a Node.js-context 📖 -> https://webpack.js.org/configuration/node/
    entry: './src/extension.ts', // the entry point of this extension, 📖 -> https://webpack.js.org/configuration/entry-context/
    output: {
      // the bundle is stored in the 'dist' folder (check package.json), 📖 -> https://webpack.js.org/configuration/output/
      path: path.resolve(__dirname, 'dist', `${targetType}`),
      filename: 'extension.js',
      library: {
        type: 'umd',
      },
      globalObject: 'globalThis',
      devtoolModuleFilenameTemplate: '../[resource-path]',
    },
    devtool: 'source-map',
    externals: {
      vscode: 'commonjs vscode', // the vscode-module is created on-the-fly and must be excluded. Add other modules that cannot be webpack'ed, 📖 -> https://webpack.js.org/configuration/externals/
    },
    resolve: {
      // support reading TypeScript and JavaScript files, 📖 -> https://github.com/TypeStrong/ts-loader
      extensions: ['.ts', '.tsx', '.js', '.jsx'],
      alias:
        targetType === 'webworker'
          ? {
              path: require.resolve('path-browserify'),
              './commands/node/inBrowserActions': path.resolve(__dirname, 'src', 'commands', 'web', 'inBrowserAction'),
              './helpers': path.resolve(__dirname, 'src', 'commands', 'web', 'helpers'),
            }
          : {
              path: require.resolve('path-browserify'),
            },
      fallback: {
        process: require.resolve('process/browser'),
        path: require.resolve('path-browserify'),
        stream: require.resolve('stream-browserify'),
        assert: require.resolve('assert'),
        util: require.resolve('util'),
      },
    },
    module: {
      rules: [
        {
          test: /\.tsx?$/,
          exclude: /node_modules/,
          use: [
            {
              loader: 'ts-loader',
            },
          ],
        },
      ],
    },
  }
}
const rootPath = path.resolve(__dirname, '../../')
const vscodeWorkspacePath = path.resolve(rootPath, 'client', 'vscode')
const vscodeSourcePath = path.resolve(vscodeWorkspacePath, 'src')
const webviewSourcePath = path.resolve(vscodeSourcePath, 'webview')

const getCSSLoaders = (...loaders) => [
  MiniCssExtractPlugin.loader,
  ...loaders,
  {
    loader: 'postcss-loader',
  },
  {
    loader: 'sass-loader',
    options: {
      sassOptions: {
        includePaths: [path.resolve(rootPath, 'node_modules'), path.resolve(rootPath, 'client')],
      },
    },
  },
]

const searchPanelWebviewPath = path.resolve(webviewSourcePath, 'search-panel')
const searchSidebarWebviewPath = path.resolve(webviewSourcePath, 'search-sidebar')
const historySidebarWebviewPath = path.resolve(webviewSourcePath, 'history-sidebar')
const extensionHostWebviewPath = path.resolve(webviewSourcePath, 'extension-host')

const extensionHostWorker = /main\.worker\.ts$/

/** @type {import('webpack').Configuration}*/

const webviewConfig = {
  name: 'webviews',
  target: 'web',
  entry: {
    searchPanel: [path.resolve(searchPanelWebviewPath, 'index.tsx')],
    searchSidebar: [path.resolve(searchSidebarWebviewPath, 'index.tsx')],
    historySidebar: [path.resolve(historySidebarWebviewPath, 'index.tsx')],
    extensionHost: [path.resolve(extensionHostWebviewPath, 'index.tsx')],
    style: path.join(webviewSourcePath, 'index.scss'),
  },
  devtool: 'source-map',
  output: {
    path: path.resolve(__dirname, 'dist/webview'),
    filename: '[name].js',
  },
  plugins: [new MiniCssExtractPlugin()],
  externals: {
    vscode: 'commonjs vscode', // the vscode-module is created on-the-fly and must be excluded. Add other modules that cannot be webpack'ed, 📖 -> https://webpack.js.org/configuration/externals/
  },
  resolve: {
    alias: {
      path: require.resolve('path-browserify'),
      './Link': path.resolve(__dirname, 'src', 'webview', 'search-panel', 'alias', 'Link'),
      '@sourcegraph/shared/src/components/Link': path.resolve(
        __dirname,
        'src',
        'webview',
        'search-panel',
        'alias',
        'Link'
      ),
      './Markdown': path.resolve(__dirname, 'src', 'webview', 'search-panel', 'alias', 'Markdown'),
      '@sourcegraph/shared/src/components/Markdown': path.resolve(
        __dirname,
        'src',
        'webview',
        'search-panel',
        'alias',
        'Markdown'
      ),
      '../documentation/ModalVideo': path.resolve(__dirname, 'src', 'webview', 'search-panel', 'alias', 'ModalVideo'),
      '@sourcegraph/branded/src/search/documentation/ModalVideo': path.resolve(
        __dirname,
        'src',
        'webview',
        'search-panel',
        'alias',
        'ModalVideo'
      ),
      './RepoFileLink': path.resolve(__dirname, 'src', 'webview', 'search-panel', 'alias', 'RepoFileLink'),
      '@sourcegraph/shared/src/components/RepoFileLink': path.resolve(
        __dirname,
        'src',
        'webview',
        'search-panel',
        'alias',
        'RepoFileLink'
      ),
      './FileMatchChildren': path.resolve(__dirname, 'src', 'webview', 'search-panel', 'alias', 'FileMatchChildren'),
      '@sourcegraph/shared/src/components/FileMatchChildren': path.resolve(
        __dirname,
        'src',
        'webview',
        'search-panel',
        'alias',
        'FileMatchChildren'
      ),
    },
    // support reading TypeScript and JavaScript files, 📖 -> https://github.com/TypeStrong/ts-loader
    extensions: ['.ts', '.tsx', '.js', '.jsx'],
    fallback: {
      path: require.resolve('path-browserify'),
      stream: require.resolve('stream-browserify'),
      process: require.resolve('process/browser'),
    },
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        exclude: [/node_modules/, extensionHostWorker],
        use: [
          {
            loader: 'ts-loader',
          },
        ],
      },
      // SCSS rule for our own styles and Bootstrap
      {
        test: /\.(css|sass|scss)$/,
        exclude: /\.module\.(sass|scss)$/,
        use: getCSSLoaders({ loader: 'css-loader', options: { url: false } }),
      },
      // For CSS modules
      {
        test: /\.(css|sass|scss)$/,
        include: /\.module\.(sass|scss)$/,
        use: getCSSLoaders({
          loader: 'css-loader',
          options: {
            sourceMap: false,
            modules: {
              exportLocalsConvention: 'camelCase',
              localIdentName: '[name]__[local]_[hash:base64:5]',
            },
            url: false,
          },
        }),
      },
      {
        test: extensionHostWorker,
        use: [
          {
            loader: 'worker-loader',
            options: { inline: 'no-fallback' },
          },
          'ts-loader',
        ],
      },
    ],
  },
}

// module.exports = [getExtensionConfig('node'), getWebviewConfig('node')]
module.exports = function () {
  return Promise.all([getExtensionConfig('node'), getExtensionConfig('webworker'), webviewConfig])
}