import { apiAuth } from "../../support/api/apiauth";
import { ensureHumanUserExists, ensureUserDoesntExist } from "../../support/api/users";
import { login, User, username } from "../../support/login/users";

describe('humans', () => {

    const humansPath = `${Cypress.env('consoleUrl')}/users/list/humans`
    const testHumanUserNameAdd = 'e2ehumanusernameadd'
    const testHumanUserNameRemove = 'e2ehumanusernameremove'

    ;[User.OrgOwner].forEach(user => {
 
        describe(`as user "${user}"`, () => {

            beforeEach(()=> {
                login(user)
                cy.visit(humansPath)
                cy.get('[data-cy=timestamp]')
            })

            describe('add', () => {
                before(`ensure it doesn't exist already`, () => {
                    apiAuth().then(apiCallProperties => {
                        ensureUserDoesntExist(apiCallProperties, testHumanUserNameAdd)
                    })
                })

                it('should add a user', () => {
                    cy.get('a[href="/users/create"]').click()
                    cy.url().should('contain', 'users/create')
                    cy.get('[formcontrolname="email"]').type(username('e2ehuman'))
                    //force needed due to the prefilled username prefix
                    cy.get('[formcontrolname="userName"]').type(testHumanUserNameAdd, {force: true})
                    cy.get('[formcontrolname="firstName"]').type('e2ehumanfirstname')
                    cy.get('[formcontrolname="lastName"]').type('e2ehumanlastname')
                    cy.get('[formcontrolname="phone"]').type('+41 123456789')
                    cy.get('[data-e2e="create-button"]').click()
                    cy.get('.data-e2e-success')
                    cy.wait(200)
                    cy.get('.data-e2e-failure', { timeout: 0 }).should('not.exist')
                })        
            })
            
            describe('remove', () => {
                before('ensure it exists', () => {
                    apiAuth().then(api => {
                        ensureHumanUserExists(api, testHumanUserNameRemove)
                    })                    
                })

                it('should delete a human user', () => {
                    cy.contains("tr", testHumanUserNameRemove, { timeout: 1000 })
                        .find('button')
                        //force due to angular hidden buttons
                        .click({force: true})
                    cy.get('[e2e-data="confirm-dialog-input"]').type(username(testHumanUserNameRemove, Cypress.env('org')))
                    cy.get('[e2e-data="confirm-dialog-button"]').click()
                    cy.get('.data-e2e-success')
                    cy.wait(200)
                    cy.get('.data-e2e-failure', { timeout: 0 }).should('not.exist')
                })
            })
        })
    })
})
/*
describe("users", ()=> {

    before(()=> {
        cy.consolelogin(Cypress.env('username'), Cypress.env('password'), Cypress.env('consoleUrl'))
    })

    it('should show personal information', () => {
        cy.log(`USER: show personal information`);
        //click on user information 
        cy.get('a[href*="users/me"').eq(0).click()
        cy.url().should('contain', '/users/me')
    })

    it('should show users', () => {
        cy.visit(Cypress.env('consoleUrl') + '/users/list/humans')
        cy.url().should('contain', 'users/list/humans')
    })
})

*/