import chalk from 'chalk';
import ora from 'ora';
import prompts from 'prompts';
import {
  checkDockerInstalled,
  isContainerRunning,
  doesContainerExist,
  runContainer,
  removeContainer,
  getAbsolutePath,
  type DockerRunOptions
} from '../utils/docker.js';

export async function startCommand(options: any) {
  console.log('\n🚀 Starting MatrixHub\n');

  // Check if Docker is installed
  const spinner = ora('Checking Docker installation...').start();
  const dockerInstalled = await checkDockerInstalled();
  
  if (!dockerInstalled) {
    spinner.fail('Docker is not installed or not in PATH');
    console.log(chalk.yellow('\nPlease install Docker first:'));
    console.log(chalk.cyan('  https://docs.docker.com/get-docker/\n'));
    process.exit(1);
  }
  spinner.succeed('Docker is installed');

  // Check if container already exists
  const containerExists = await doesContainerExist(options.name);
  const containerRunning = await isContainerRunning(options.name);

  if (containerRunning) {
    console.log(chalk.yellow(`\n⚠️  Container "${options.name}" is already running`));
    console.log(chalk.cyan(`   Access MatrixHub at http://localhost:${options.port}\n`));
    process.exit(0);
  }

  if (containerExists) {
    const response = await prompts({
      type: 'confirm',
      name: 'remove',
      message: `Container "${options.name}" already exists. Remove and recreate?`,
      initial: true
    });

    if (!response.remove) {
      console.log(chalk.yellow('\nOperation cancelled\n'));
      process.exit(0);
    }

    const removeSpinner = ora('Removing existing container...').start();
    await removeContainer(options.name);
    removeSpinner.succeed('Container removed');
  }

  // Convert data path to absolute path
  const absoluteDataPath = getAbsolutePath(options.data);

  // Pull image
  const pullSpinner = ora(`Pulling Docker image: ${options.image}...`).start();
  try {
    await import('../utils/docker.js').then(m => m.pullImage(options.image));
    pullSpinner.succeed('Docker image pulled');
  } catch (error) {
    pullSpinner.warn('Failed to pull image, will use local image if available');
  }

  // Start container
  const startSpinner = ora('Starting MatrixHub container...').start();
  try {
    const runOptions: DockerRunOptions = {
      name: options.name,
      image: options.image,
      port: options.port,
      dataPath: absoluteDataPath,
      detach: options.detach
    };

    await runContainer(runOptions);
    startSpinner.succeed('MatrixHub container started');

    console.log(chalk.green.bold('\n✅ MatrixHub is now running!\n'));
    console.log(chalk.cyan('📍 Access MatrixHub at:'), chalk.bold(`http://localhost:${options.port}`));
    console.log(chalk.cyan('📁 Data directory:'), chalk.bold(absoluteDataPath));
    console.log(chalk.cyan('🐳 Container name:'), chalk.bold(options.name));
    console.log(chalk.gray('\n💡 Useful commands:'));
    console.log(chalk.gray('   matrixhub status   - Check container status'));
    console.log(chalk.gray('   matrixhub logs     - View container logs'));
    console.log(chalk.gray('   matrixhub stop     - Stop the container'));
    console.log(chalk.gray('   matrixhub restart  - Restart the container\n'));
  } catch (error: any) {
    startSpinner.fail('Failed to start container');
    console.error(chalk.red('\nError:'), error.message);
    process.exit(1);
  }
}
