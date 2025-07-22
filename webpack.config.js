const path = require('path');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const TerserPlugin = require('terser-webpack-plugin');
const CssMinimizerPlugin = require('css-minimizer-webpack-plugin');

const isProduction = process.env.NODE_ENV === 'production';

module.exports = {
  mode: isProduction ? 'production' : 'development',
  
  entry: {
    // Main application bundle
    app: {
      import: './src/js/app.js',
      dependOn: 'vendor'
    },
    
    // Vendor libraries bundle
    vendor: [
      './static/accessible_forms.js',
      './static/enhanced_help_system.js',
      './static/charts.js',
      './static/logger.js'
    ],
    
    // Wizard-specific bundle
    wizards: {
      import: [
        './static/import_wizard.js',
        './static/maintenance_wizard.js',
        './static/route_assignment_wizard.js',
        './static/wizard.js'
      ],
      dependOn: 'vendor'
    },
    
    // Main styles
    styles: './src/css/main.css'
  },
  
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: isProduction ? '[name].[contenthash].js' : '[name].js',
    clean: true,
    publicPath: '/dist/'
  },
  
  module: {
    rules: [
      // JavaScript processing
      {
        test: /\.js$/,
        exclude: /node_modules/,
        use: {
          loader: 'babel-loader',
          options: {
            presets: [
              ['@babel/preset-env', {
                targets: {
                  browsers: ['> 1%', 'last 2 versions']
                }
              }]
            ]
          }
        }
      },
      
      // CSS processing
      {
        test: /\.css$/,
        use: [
          MiniCssExtractPlugin.loader,
          'css-loader',
          'postcss-loader'
        ]
      },
      
      // Asset processing
      {
        test: /\.(png|svg|jpg|jpeg|gif|ico)$/i,
        type: 'asset/resource',
        generator: {
          filename: 'images/[name].[contenthash][ext]'
        }
      },
      
      {
        test: /\.(woff|woff2|eot|ttf|otf)$/i,
        type: 'asset/resource',
        generator: {
          filename: 'fonts/[name].[contenthash][ext]'
        }
      }
    ]
  },
  
  plugins: [
    new MiniCssExtractPlugin({
      filename: isProduction ? '[name].[contenthash].css' : '[name].css'
    })
  ],
  
  optimization: {
    minimize: isProduction,
    minimizer: [
      new TerserPlugin({
        terserOptions: {
          compress: {
            drop_console: isProduction
          }
        }
      }),
      new CssMinimizerPlugin()
    ],
    
    splitChunks: {
      cacheGroups: {
        vendor: {
          test: /[\\/]node_modules[\\/]/,
          name: 'vendors',
          chunks: 'all'
        }
      }
    }
  },
  
  devtool: isProduction ? 'source-map' : 'eval-source-map',
  
  watchOptions: {
    ignored: /node_modules/,
    poll: 1000
  }
};