const path = require('path');
const CopyWebpackPlugin = require('copy-webpack-plugin');

let imageSizeLimit = 9007199254740991; // Number.MAX_SAFE_INTEGER
let sourceDir = path.resolve(__dirname, 'src');
let buildDir = path.resolve(__dirname, 'build');

module.exports = {
	entry: {
		index: path.resolve(sourceDir, 'main.js')
	},
	output: {
		path: buildDir,
		filename: 'main.js'
	},
	optimization: {
		splitChunks: false
	},
	performance: {
		maxEntrypointSize: 100000000,
		maxAssetSize: 100000000,
		hints: 'warning'
	},
	devServer: {
		allowedHosts: 'all',
		static: path.join(__dirname, 'src'),
		compress: true,
		open: false,
		port: 9901,
		hot: true,
		historyApiFallback: true
	},
	mode: 'development',
	cache: {
		type: 'filesystem'
	},
	devtool: 'cheap-module-source-map',
	resolve: {
		extensions: ['.js', '.jsx', '.json']
	},
	module: {
		rules: [
			{
				test: /\.(js|jsx)$/,
				exclude: /node_modules/,
				use: {
					loader: 'babel-loader',
					options: {
						presets: ['@babel/preset-env', '@babel/preset-react']
					}
				}
			},
			{
				test: /\.css$/,
				use: [
					'style-loader',
					'css-loader',
					'postcss-loader'
				]
			},
			{
				test: /\.(png|gif|jpg|woff2?|eot|ttf|otf|svg)(\?.*)?$/i,
				use: [
					{
						loader: 'url-loader',
						options: {
							limit: imageSizeLimit
						}
					}
				],
			}
		]
	},
	plugins: [
		new CopyWebpackPlugin({
			patterns: [
				{
					from: path.resolve(sourceDir, 'index.html'),
					to: path.resolve(buildDir, 'index.html')
				},
			]
		})
	]
};
