<script>
	let name = 'CodeAtlas';
	let repositoryUrl = '';
	let serverUrl = 'http://localhost:8080';
	let uploading = false;
	let message = '';

	async function uploadRepository() {
		if (!repositoryUrl) {
			message = 'Please enter a repository URL';
			return;
		}

		uploading = true;
		message = 'Uploading repository...';

		try {
			// In a real implementation, this would call the backend API
			// For now, we'll simulate the upload
			await new Promise(resolve => setTimeout(resolve, 2000));
			
			message = 'Repository uploaded successfully!';
		} catch (error) {
			message = 'Error uploading repository: ' + error.message;
		} finally {
			uploading = false;
		}
	}
</script>

<main>
	<h1>Hello {name}!</h1>
	
	<div class="upload-form">
		<h2>Upload Repository</h2>
		<form on:submit|preventDefault={uploadRepository}>
			<div>
				<label for="repositoryUrl">Repository URL:</label>
				<input 
					id="repositoryUrl" 
					type="text" 
				 bind:value={repositoryUrl} 
					placeholder="https://github.com/user/repo"
					disabled={uploading}
				/>
			</div>
			
			<div>
				<label for="serverUrl">Server URL:</label>
				<input 
					id="serverUrl" 
					type="text" 
					bind:value={serverUrl} 
					placeholder="http://localhost:8080"
					disabled={uploading}
				/>
			</div>
			
			<button type="submit" disabled={uploading}>
				{uploading ? 'Uploading...' : 'Upload Repository'}
			</button>
		</form>
		
		{#if message}
			<p class="message">{message}</p>
		{/if}
	</div>
</main>

<style>
	main {
		text-align: center;
		padding: 1em;
		max-width: 240px;
		margin: 0 auto;
	}

	h1 {
		color: #ff3e00;
		text-transform: uppercase;
		font-size: 4em;
		font-weight: 100;
	}

	h2 {
		color: #666;
	}

	.upload-form {
		background: #f5f5f5;
		padding: 1em;
		border-radius: 5px;
		margin-top: 2em;
	}

	form div {
		margin-bottom: 1em;
		text-align: left;
	}

	label {
		display: block;
		margin-bottom: 0.5em;
		font-weight: bold;
	}

	input {
		width: 100%;
		padding: 0.5em;
		border: 1px solid #ccc;
		border-radius: 3px;
	}

	button {
		background: #ff3e00;
		color: white;
		border: none;
		padding: 0.75em 1.5em;
		border-radius: 3px;
		cursor: pointer;
		font-size: 1em;
	}

	button:disabled {
		background: #ccc;
		cursor: not-allowed;
	}

	.message {
		margin-top: 1em;
		padding: 0.5em;
		border-radius: 3px;
	}

	.message.error {
		background: #ffebee;
		color: #c62828;
	}
</style>