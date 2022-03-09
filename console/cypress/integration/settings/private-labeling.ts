import { apiAuth, apiCallProperties } from "../../support/api/apiauth";
import { Policy, resetPolicy } from "../../support/api/policies";
import { login, User } from "../../support/login/users";

describe("private labeling", ()=> {

    const orgPath = `${Cypress.env('consoleUrl')}/org`

    ;[User.OrgOwner].forEach(user => {

        describe(`as user "${user}"`, () => {

            let api: apiCallProperties


            beforeEach(()=> {
                login(user)
                cy.visit(orgPath)
                // TODO: Why force?
                cy.contains('[data-e2e=policy-card]', 'Private Labeling').contains('button', 'Modify').click({force: true}) // TODO: select data-e2e
            })
                        
            customize('white', user)
            customize('dark', user)
        })
    })
})


function customize(theme: string, user: User) {

    describe(`${theme} theme`, () => {

        beforeEach(() => {
            apiAuth().then(api => {
                resetPolicy(api, Policy.Label)
            })
        })

        describe.skip('logo', () => {

            beforeEach('expand logo category', () => {
                cy.contains('[data-e2e=policy-category]', 'Logo').click() // TODO: select data-e2e
                cy.fixture('logo.png').as('logo')
            })

            it('should update a logo', () => {
                cy.get('[data-e2e=image-part-logo]').find('input').then(function (el) {
                    const blob = Cypress.Blob.base64StringToBlob(this.logo, 'image/png')
                    const file = new File([blob], 'images/logo.png', { type: 'image/png' })
                    const list = new DataTransfer()
                
                    list.items.add(file)
                    const myFileList = list.files
                
                    el[0].files = myFileList
                    el[0].dispatchEvent(new Event('change', { bubbles: true }))
                })
            })
            it('should delete a logo')
        })
        it('should update an icon')
        it('should delete an icon')
        it.skip('should update the background color', () => {
            cy.contains('[data-e2e=color]', 'Background Color').find('button').click() // TODO: select data-e2e
            cy.get('color-editable-input').find('input').clear().type('#ae44dc')
            cy.get('[data-e2e=save-colors-button]').click()
            cy.get('[data-e2e=header-user-avatar]').click()
            cy.contains('Logout All Users').click() // TODO: select data-e2e
            login(User.LoginPolicyUser, true, null, () => {
                cy.pause()
            })
        })
        it('should update the primary color')
        it('should update the warning color')
        it('should update the font color')
        it('should update the font style')
        it('should hide the loginname suffix')
        it('should show the loginname suffix')
        it('should hide the watermark')
        it('should show the watermark')
        it('should show the current configuration')
        it('should reset the policy')
    })
}
