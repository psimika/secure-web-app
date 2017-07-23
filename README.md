# Secure web app

Implementation of a secure web application.

## Development

### Running the app locally using Heroku

Create a .env.local file with the following key value pairs:

    DATABASE_URL="user=<db user> password=<db password> dbname=petfind"
    TMPL_PATH="web"
    PORT="8080"

Then install and run with:

    go install -tags=heroku ./... && heroku local -e .env.local

Alternatively, in order to run the heroku version without using the heroku command:

    go install -tags=heroku ./...

    TMPL_PATH=web PORT=8080 DATABASE_URL="user=<db user> password=<db password> database=petfind" petfindserver

## Deployment

### On Heroku

In order to deploy to Heroku for the first time we need these steps:

    heroku login

    heroku create

    heroku addons:create heroku-postgresql:hobby-dev

After that and each time we make a change on master branch:

    git push heroku master

Or when working on a different branch:

    git push heroku somebranch:master

### On a Linux server (Ubuntu 16.04 example)

First we connect to the server and create a `petfind` account:

    sudo adduser petfind

And we install PostgreSQL:

    sudo apt-get install postgresql postgresql-contrib

Then we create a PostgreSQL user and database `petfind`. There is no need for
the database user to be a superuser or have other priviledges.

    sudo -u postgres createuser --interactive
    sudo -u postgres createdb petfind

We may also want to add a password to the database user. First we connect to
PostgreSQL with:

    sudo -u postgres psql

Then we alter the user so that it has a password by using the following SQL
statement:

    ALTER USER petfind WITH password '<db password>';

We upload the `petfindserver` binary to the server in `/home/petfind`. The
application's templates that exist under the code's `web/templates` should also
be uploaded in `/home/petfind/templates`. If we are planning to provide our own
SSL certificates, they should also be uploaded in `/home/petfind`.

Normally it is not allowed for programs to access system ports than are less
than 1024. In order for the system to allow `petfindserver` to listen to ports
`80` and `443` which are less than 1024, we need to give it special permissions
(Lee 2017).

    sudo setcap 'cap_net_bind_service=+ep' /home/petfind/petfindserver

Ubuntu 16.04 uses Systemd for managing services. We need a Systemd file that
describes our service. We create a `[petfind.service](doc/petfind.service)`
file in `/etc/systemd/system/` which looks like this:

    [Unit]
    Description=petfind server
    ConditionPathExists=/home/petfind
    After=network.target

    [Service]
    Type=simple
    User=petfind
    Group=petfind
    Restart=always
    RestartSec=10
    StartLimitIntervalSec=60
    WorkingDirectory=/home/petfind

    # Automatic Let's Encrypt certificates example
    ExecStart=/home/petfind/petfindserver -http=:80 -https=:443 -tmpl=/home/petfind -datasource="user=petfind password=<db password> dbname=petfind" -autocert=petfind.example.com -autocertdir=/home/petfind/letscache

    # Provided certificates example
    # ExecStart=/home/petfind/petfindserver -http=:80 -https=:443 -tmpl=/home/petfind -datasource="user=petfind password=<db password> dbname=petfind" -tlscert=/home/petfind/cert.pem -tlskey=/home/petfind/key.pem

    # Insecure Example
    # ExecStart=/home/petfind/petfindserver -http=:80 -tmpl=/home/petfind -datasource="user=petfind password=<db password> dbname=petfind" -insecure

    [Install]
    WantedBy=multi-user.target

The above file contains three examples of flag usage for the application. Only
one of them should be left uncommented depending on how we wish to run the
application.

The application supports fetching Let's Encrypt certificates automatically for
one or more domains. For this case we use the flag `-autocert="<domains>"` and
we provide one or more domains (separated by spaces) so for example if the
server's domain is petfind.example.com we provide that as a value. It is
recommended to cache the Let's Encrypt certificates somewhere otherwise the
application will have to request them again when it restarts. For this case we
provide a folder for the certificates to be cached with the flag
`-autocertdir=/home/petfind/letscache`.

If we wish to provide our own certificates, we instead use the flags `-tlscert`
and `-tlskey` providing the public and private keys respectively in PEM format.

As a last option, there is the `-insecure` flag which forces the application to
only serve insecure HTTP instead of HTTPS.

After we are done, we can enable the service with:

    sudo systemctl enable petfind.service

If we make any changes to the service file we need to use:

    sudo systemctl daemon-reload

We can start and stop the service with:

    sudo systemctl start petfind.service
    sudo systemctl stop petfind.service

Finally we can inspect the logs to make sure that the petfind server has
started serving:

    sudo tail -f /var/log/syslog

## References

Beams, C. (2014). *How to Write a Git Commit Message* [online] Available at: https://chris.beams.io/posts/git-commit/ [Accessed: June 28 2017]

Bourgon, P. (2014). *Go: Best Practices for Production Environments* [online] Available at: http://peter.bourgon.org/go-in-production/#testing-and-validation [Accessed: July 18 2017]

Edwards, A. (2015). *Practical Persistence in Go: Organising Database Access* [online] Available at: http://www.alexedwards.net/blog/organising-database-access [Accessed: July 18 2017]

Gerrand, A. (2011a). *Error handling and Go* [online] Available at: https://blog.golang.org/error-handling-and-go [Accessed: July 19 2017]

Gerrand, A. (2011b). *Godoc: documenting Go code* [online] Available at: https://blog.golang.org/godoc-documenting-go-code [Accessed: July 19 2017]

Gerrand, A. (2012). *Organizing Go code* [online] Available at: https://blog.golang.org/organizing-go-code [Accessed: July 18 2017]

Heroku Dev Center (2017). *HTTP Routing* [online] Available at: https://devcenter.heroku.com/articles/http-routing#heroku-headers [Accessed: July 21 2017]

Johnson, B. (2014). *Structuring Applications in Go* [online] Available at: https://medium.com/@benbjohnson/structuring-applications-in-go-3b04be4ff091 [Accessed: July 18 2017]

Johnson, B. (2016). *Standard Package Layout* [online] Available at: https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1 [Accessed: July 18 2017]

Lee, F. (2017). *GoLang: Running a Go binary as a systemd service on Ubuntu 16.04* [online] Available at: https://fabianlee.org/2017/05/21/golang-running-a-go-binary-as-a-systemd-service-on-ubuntu-16-04/ [Accessed: July 23 2017]

OWASP (2017). *HTTP Strict Transport Security Cheat Sheet* [online] Available at: https://www.owasp.org/index.php/HTTP_Strict_Transport_Security_Cheat_Sheet [Accessed: July 19 2017]

Pike, R. (2013). *The cover story* [online] Available at: https://blog.golang.org/cover [Accessed: July 18 2017]

Valsorda, F. (2016). *So you want to expose Go on the Internet*  [online] Available at: https://blog.cloudflare.com/exposing-go-on-the-internet/ [Accessed: July 19 2017]
