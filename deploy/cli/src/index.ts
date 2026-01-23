#!/usr/bin/env node

import { Command } from 'commander';
import { startCommand } from './commands/start.js';
import { stopCommand } from './commands/stop.js';
import { restartCommand } from './commands/restart.js';
import { statusCommand } from './commands/status.js';
import { logsCommand } from './commands/logs.js';
import { updateCommand } from './commands/update.js';

const LOGO = `
  __  __       _        _      _   _       _     
 |  \\/  | __ _| |_ _ __(_)_  _| | | |_   _| |__  
 | |\\/| |/ _\` | __| '__| \\ \\/ / |_| | | | | '_ \\ 
 | |  | | (_| | |_| |  | |>  <|  _  | |_| | |_) |
 |_|  |_|\\__,_|\\__|_|  |_/_/\\_\\_| |_|\\__,_|_.__/ 
`;

const program = new Command();

program
  .name('matrixhub')
  .usage('[options] [command]')
  .description('CLI tool for deploying and managing MatrixHub')
  .version('0.1.4', '-v, --version', 'Show version and exit')
  .addHelpText('beforeAll', LOGO + '\nMatrixHub CLI - Self-hosted AI model registry\n');

program
  .command('start')
  .description('Start MatrixHub container')
  .option('-p, --port <port>', 'Port to expose (default: 9527)', '9527')
  .option('-d, --data <path>', 'Data directory path (default: ./data)', './data')
  .option('-n, --name <name>', 'Container name (default: matrixhub)', 'matrixhub')
  .option('--image <image>', 'Docker image to use', 'ghcr.io/matrixhub-ai/matrixhub:main')
  .option('--detach', 'Run container in detached mode', true)
  .action(startCommand);

program
  .command('stop')
  .description('Stop MatrixHub container')
  .option('-n, --name <name>', 'Container name (default: matrixhub)', 'matrixhub')
  .action(stopCommand);

program
  .command('restart')
  .description('Restart MatrixHub container')
  .option('-n, --name <name>', 'Container name (default: matrixhub)', 'matrixhub')
  .action(restartCommand);

program
  .command('status')
  .description('Check MatrixHub container status')
  .option('-n, --name <name>', 'Container name (default: matrixhub)', 'matrixhub')
  .action(statusCommand);

program
  .command('logs')
  .description('View MatrixHub container logs')
  .option('-n, --name <name>', 'Container name (default: matrixhub)', 'matrixhub')
  .option('-f, --follow', 'Follow log output', false)
  .option('--tail <lines>', 'Number of lines to show from the end of the logs', '100')
  .action(logsCommand);

program
  .command('update')
  .description('Update MatrixHub to the latest version')
  .option('-n, --name <name>', 'Container name (default: matrixhub)', 'matrixhub')
  .option('--image <image>', 'Docker image to use', 'ghcr.io/matrixhub-ai/matrixhub:main')
  .action(updateCommand);

program.parse();
