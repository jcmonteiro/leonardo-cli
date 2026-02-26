# leonardo-cli

`leonardo-cli` is a simple command‑line tool written in Go that wraps parts of the [Leonardo.Ai API](https://docs.leonardo.ai/).  It allows you to kick off image generation jobs and poll their status from your own scripts or terminals.  This first version implements two primary commands:

* `create` — Start a new text‑to‑image generation.  You can specify your prompt and optional parameters like model ID, image dimensions, number of images and more.
* `status` — Check the progress of a previously started generation using its ID.  This command reports the status and prints any available image URLs once the job is complete.

## Requirements

* Go 1.20 or later to build from source.
* A Leonardo.Ai API key with sufficient credits.  Set the key in your environment as `LEONARDO_API_KEY` before running the CLI.  You can obtain a key from your account’s **API Access** page on Leonardo.Ai.

## Installation

Clone the repository and build the binary:

```sh
git clone <repo-url> leonardo-cli
cd leonardo-cli
go build -o leonardo ./cmd/leonardo
```

Make sure your `LEONARDO_API_KEY` environment variable is exported:

```sh
export LEONARDO_API_KEY="your‑api‑key-here"
```

## Usage

Run `leonardo` without any arguments to see the available commands:

```sh
./leonardo
```

### Create a generation

The `create` command submits a new image generation request.  A prompt is required.  Optional flags let you control the model, resolution and other parameters.  For example:

```sh
./leonardo create \
  --prompt "A serene watercolor painting of a mountain lake at sunrise" \
  --model-id 7b592283-e8a7-4c5a-9ba6-d18c31f258b9 \
  --width 1920 \
  --height 1080 \
  --num-images 4 \
  --style-uuid 111dc692-d470-4eec-b791-3475abac4c46 \
  --contrast 3.5 \
  --alchemy=false \
  --ultra=false
```

If the call is successful, the CLI prints the returned `generationId` along with the full JSON response.  The generation ID can be used to poll for status.

In the [Quick Start Guide](https://docs.leonardo.ai/docs/getting-started), Leonardo explains that after submitting a generation you receive an identifier (often called `generationId`) that is used in subsequent calls【202409399148263†L150-L176】.

### Check generation status

Use the `status` command with the generation ID to check if your images are ready:

```sh
./leonardo status --id 123456-0987-aaaa-bbbb-01010101010
```

The CLI will query `GET /api/rest/v1/generations/{id}`.  It attempts to extract the `status` and any image URLs from the response.  If the status is `PENDING`, no images will be returned.  According to Leonardo’s API FAQs, the `generated_images` array remains empty until the job is complete【271928005095238†L183-L201】.  Once the status is `COMPLETE`, the response contains URLs to the generated images.

The full JSON response is printed for completeness.

## Notes

* Only the most common parameters are exposed as flags.  Refer to the official documentation for advanced options such as `guidance_scale`, `init_image_id` and ControlNet parameters.
* The CLI does not implement any retry logic.  For long‑running jobs you may need to poll repeatedly until the status changes to `COMPLETE`【271928005095238†L183-L201】.
* Ensure your API key has sufficient credits.  The API will return an error if your credit balance is low.

## License

This project is provided under the MIT License.  See `LICENSE` for details.