const webpack = require("webpack");

module.exports = {
	mode: 'development',
  entry: "./entry.point",
  output: {
		path: __dirname,
    filename: "testutils.inc.js",
    libraryTarget: "this",
  }
};
