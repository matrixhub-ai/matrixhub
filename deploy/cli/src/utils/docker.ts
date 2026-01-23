import { exec } from 'child_process';
import { promisify } from 'util';
import chalk from 'chalk';

const execAsync = promisify(exec);

export interface DockerRunOptions {
  name: string;
  image: string;
  port: string;
  dataPath: string;
  detach: boolean;
}

export async function checkDockerInstalled(): Promise<boolean> {
  try {
    await execAsync('docker --version');
    return true;
  } catch (error) {
    return false;
  }
}

export async function isContainerRunning(name: string): Promise<boolean> {
  try {
    const { stdout } = await execAsync(
      `docker ps --filter "name=${name}" --format "{{.Names}}"`
    );
    return stdout.trim() === name;
  } catch (error) {
    return false;
  }
}

export async function doesContainerExist(name: string): Promise<boolean> {
  try {
    const { stdout } = await execAsync(
      `docker ps -a --filter "name=${name}" --format "{{.Names}}"`
    );
    return stdout.trim() === name;
  } catch (error) {
    return false;
  }
}

export async function runContainer(options: DockerRunOptions): Promise<void> {
  const { name, image, port, dataPath, detach } = options;
  
  const cmd = [
    'docker run',
    detach ? '-d' : '',
    `--name ${name}`,
    `--restart unless-stopped`,
    `-p ${port}:9527`,
    `-v ${dataPath}:/data`,
    image
  ].filter(Boolean).join(' ');

  await execAsync(cmd);
}

export async function stopContainer(name: string): Promise<void> {
  await execAsync(`docker stop ${name}`);
}

export async function removeContainer(name: string): Promise<void> {
  await execAsync(`docker rm ${name}`);
}

export async function restartContainer(name: string): Promise<void> {
  await execAsync(`docker restart ${name}`);
}

export async function getContainerStatus(name: string): Promise<string> {
  try {
    const { stdout } = await execAsync(
      `docker ps -a --filter "name=${name}" --format "{{.Status}}"`
    );
    return stdout.trim();
  } catch (error) {
    return 'Not found';
  }
}

export async function getContainerLogs(
  name: string,
  follow: boolean = false,
  tail: string = '100'
): Promise<void> {
  const cmd = [
    'docker logs',
    follow ? '-f' : '',
    `--tail ${tail}`,
    name
  ].filter(Boolean).join(' ');

  if (follow) {
    // For follow mode, we need to use spawn instead of exec
    const { spawn } = await import('child_process');
    const child = spawn('docker', ['logs', '-f', '--tail', tail, name], {
      stdio: 'inherit'
    });

    process.on('SIGINT', () => {
      child.kill('SIGTERM');
      process.exit(0);
    });
  } else {
    const { stdout } = await execAsync(cmd);
    console.log(stdout);
  }
}

export async function pullImage(image: string): Promise<void> {
  await execAsync(`docker pull ${image}`);
}

export function getAbsolutePath(path: string): string {
  if (path.startsWith('/')) {
    return path;
  }
  if (path.startsWith('~/')) {
    return path.replace('~', process.env.HOME || '~');
  }
  return `${process.cwd()}/${path}`;
}
