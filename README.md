# github-activity-cli

This project is a sample solution for the [Github Activity](https://roadmap.sh/projects/github-user-activity) challenge from [roadmap.sh](https://roadmap.sh/).

## How to run

Clone the repository and run the following command:

```bash
git clone https://github.com/febryansambuari/github-activity-cli.git
cd github-activity-cli
```

Run the following command to build and run the project:

```bash
# Build the project
go build -o github-activity-cli

# Fetch the github events
./github-activity-cli [github username]
example: ./github-activity-cli febryansambuari
```

in this project, I also added a simple caching technique to store a file cache.
