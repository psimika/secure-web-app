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

In order to deploy to Heroku for the first time we need these steps:

    heroku login

    heroku create

    heroku addons:create heroku-postgresql:hobby-dev

After that and each time we make a change on master branch:

    git push heroku master

Or when working on a different branch:

    git push heroku somebranch:master

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

OWASP (2017). *HTTP Strict Transport Security Cheat Sheet* [online] Available at: https://www.owasp.org/index.php/HTTP_Strict_Transport_Security_Cheat_Sheet [Accessed: July 19 2017]

Pike, R. (2013). *The cover story* [online] Available at: https://blog.golang.org/cover [Accessed: July 18 2017]

Valsorda, F. (2016). *So you want to expose Go on the Internet*  [online] Available at: https://blog.cloudflare.com/exposing-go-on-the-internet/ [Accessed: July 19 2017]
